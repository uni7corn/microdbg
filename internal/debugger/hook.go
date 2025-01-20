package debugger

import (
	"slices"
	"sync"

	"github.com/wnxd/microdbg/debugger"
	"github.com/wnxd/microdbg/emulator"
)

type hookManger struct {
	releases  []func() error
	ctrlAddrs []chan [2]uint64
	mu        sync.Mutex
	intrHooks []debugger.HookHandler
	insnHooks []debugger.HookHandler
	memHooks  []debugger.HookHandler
}

type hookHandler[T any] struct {
	releases   []func() error
	typ        emulator.HookType
	callback   T
	data       any
	begin, end uint64
}

type codeHandler struct {
	hookHandler[debugger.CodeCallback]
}

type memHandler struct {
	hookHandler[debugger.MemoryCallback]
}

type controlHandler struct {
	releases []func() error
	addr     [2]uint64
	callback debugger.ControlCallback
}

func (h *hookManger) ctor(dbg Debugger) {
	if hook, err := dbg.Emulator().Hook(emulator.HOOK_TYPE_INTR, h.handleInterrupt, dbg, 1, 0); err == nil {
		h.releases = append(h.releases, hook.Close)
	}
	if hook, err := dbg.Emulator().Hook(emulator.HOOK_TYPE_INSN_INVALID, h.handleInvalid, dbg, 1, 0); err == nil {
		h.releases = append(h.releases, hook.Close)
	}
	if hook, err := dbg.Emulator().Hook(emulator.HOOK_TYPE_MEM_INVALID, h.handleMemory, dbg, 1, 0); err == nil {
		h.releases = append(h.releases, hook.Close)
	}
}

func (h *hookManger) dtor() {
	for i := len(h.releases) - 1; i >= 0; i-- {
		h.releases[i]()
	}
	h.releases = nil
}

func (h *hookManger) allocCtrlAddrs(dbg Debugger) error {
	region, err := dbg.MapAlloc(0x1000, emulator.MEM_PROT_EXEC)
	if err != nil {
		return err
	}
	h.releases = append(h.releases, func() error {
		return dbg.MapFree(region.Addr, region.Size)
	})
	emu := dbg.Emulator()
	var asm []byte
	switch emu.Arch() {
	case emulator.ARCH_ARM:
		asm = []byte{0x35, 0x00, 0x00, 0xef}
	case emulator.ARCH_ARM64:
		asm = []byte{0xa1, 0x06, 0x00, 0xd4}
	case emulator.ARCH_X86, emulator.ARCH_X86_64:
		// asm = []byte{0x90}
	}
	size := uint64(len(asm))
	count := region.Size / size
	end := region.Addr + (size * count)
	ch := make(chan [2]uint64, count)
	for addr := region.Addr; addr < end; addr += size {
		emu.MemWrite(addr, asm)
		ch <- [2]uint64{addr, addr + size}
	}
	h.ctrlAddrs = append(h.ctrlAddrs, ch)
	return nil
}

func (h *hookManger) ctrlAddrAlloc(dbg Debugger) (addr [2]uint64, err error) {
	for {
		for _, ch := range h.ctrlAddrs {
			select {
			case addr = <-ch:
				return
			default:
			}
		}
		err = h.allocCtrlAddrs(dbg)
		if err != nil {
			return
		}
	}
}

func (h *hookManger) ctrlAddrFree(addr [2]uint64) {
	for _, ch := range h.ctrlAddrs {
		select {
		case ch <- addr:
			return
		default:
		}
	}
}

func (h *hookManger) addHook(dbg Debugger, typ emulator.HookType, callback any, data any, begin, end uint64) (debugger.HookHandler, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	switch typ {
	case emulator.HOOK_TYPE_INTR:
		callback, ok := callback.(debugger.InterruptCallback)
		if !ok {
			return nil, debugger.ErrHookCallbackType
		}
		handler := &hookHandler[debugger.InterruptCallback]{typ: typ, callback: callback, data: data, begin: begin, end: end}
		h.intrHooks = append(h.intrHooks, handler)
		handler.releases = append(handler.releases, func() error {
			h.mu.Lock()
			h.intrHooks = slices.DeleteFunc(h.intrHooks, func(i debugger.HookHandler) bool { return i == handler })
			h.mu.Unlock()
			return nil
		})
		return handler, nil
	case emulator.HOOK_TYPE_INSN_INVALID:
		callback, ok := callback.(debugger.InvalidCallback)
		if !ok {
			return nil, debugger.ErrHookCallbackType
		}
		handler := &hookHandler[debugger.InvalidCallback]{typ: typ, callback: callback, data: data, begin: begin, end: end}
		h.insnHooks = append(h.insnHooks, handler)
		handler.releases = append(handler.releases, func() error {
			h.mu.Lock()
			h.insnHooks = slices.DeleteFunc(h.insnHooks, func(i debugger.HookHandler) bool { return i == handler })
			h.mu.Unlock()
			return nil
		})
		return handler, nil
	case emulator.HOOK_TYPE_CODE, emulator.HOOK_TYPE_BLOCK:
		callback, ok := callback.(debugger.CodeCallback)
		if !ok {
			return nil, debugger.ErrHookCallbackType
		}
		handler := &codeHandler{hookHandler: hookHandler[debugger.CodeCallback]{typ: typ, callback: callback, data: data}}
		hook, err := dbg.Emulator().Hook(typ, handler.handleCode, dbg, begin, end)
		if err != nil {
			return nil, err
		}
		handler.releases = append(handler.releases, hook.Close)
		return handler, nil
	default:
		callback, ok := callback.(debugger.MemoryCallback)
		if !ok {
			return nil, debugger.ErrHookCallbackType
		}
		handler := &memHandler{hookHandler: hookHandler[debugger.MemoryCallback]{typ: typ, callback: callback, data: data, begin: begin, end: end}}
		if invalid := typ & emulator.HOOK_TYPE_MEM_INVALID; invalid != 0 {
			h.memHooks = append(h.memHooks, handler)
			handler.releases = append(handler.releases, func() error {
				h.mu.Lock()
				h.memHooks = slices.DeleteFunc(h.memHooks, func(i debugger.HookHandler) bool { return i == handler })
				h.mu.Unlock()
				return nil
			})
		}
		if valid := typ & (emulator.HOOK_TYPE_MEM_VALID | emulator.HOOK_TYPE_MEM_READ_AFTER); valid != 0 {
			hook, err := dbg.Emulator().Hook(typ, handler.handleMemory, dbg, begin, end)
			if err != nil {
				handler.Close()
				return nil, err
			}
			handler.releases = append(handler.releases, hook.Close)
		}
		return handler, nil
	}
}

func (h *hookManger) addControl(dbg Debugger, callback debugger.ControlCallback, data any) (debugger.ControlHandler, error) {
	addr, err := h.ctrlAddrAlloc(dbg)
	if err != nil {
		return nil, err
	}
	handler := &controlHandler{
		addr:     addr,
		callback: callback,
	}
	hook, err := h.addHook(dbg, emulator.HOOK_TYPE_INTR, handler.handleControl, data, addr[1], addr[1]+1)
	if err != nil {
		h.ctrlAddrFree(addr)
		return nil, err
	}
	handler.releases = append(handler.releases, hook.Close, func() error {
		h.ctrlAddrFree(addr)
		return nil
	})
	return handler, nil
}

func (h *hookManger) handleInterrupt(intno uint64, data any) {
	data.(Debugger).asyncTask(func(task debugger.Task) {
		result := debugger.HookResult_Next
		ctx := task.Context()
		for _, hook := range h.intrHooks {
			handler := hook.(*hookHandler[debugger.InterruptCallback])
			if handler.valid(emulator.HOOK_TYPE_INTR, ctx) {
				result = handler.callback(ctx, intno, handler.data)
				if result == debugger.HookResult_Done {
					break
				}
			}
		}
		if result == debugger.HookResult_Next {
			task.CancelCause(debugger.NewInterruptException(ctx, intno))
		}
	})
}

func (h *hookManger) handleInvalid(data any) bool {
	data.(Debugger).asyncTask(func(task debugger.Task) {
		result := debugger.HookResult_Next
		ctx := task.Context()
		for _, hook := range h.insnHooks {
			handler := hook.(*hookHandler[debugger.InvalidCallback])
			if handler.valid(emulator.HOOK_TYPE_INSN_INVALID, ctx) {
				result = handler.callback(ctx, handler.data)
				if result == debugger.HookResult_Done {
					break
				}
			}
		}
		if result == debugger.HookResult_Next {
			task.CancelCause(debugger.NewInvalidInstructionException(ctx))
		}
	})
	return true
}

func (h *hookManger) handleMemory(typ emulator.HookType, addr, size, value uint64, data any) bool {
	data.(Debugger).asyncTask(func(task debugger.Task) {
		result := debugger.HookResult_Next
		ctx := task.Context()
		for _, hook := range h.memHooks {
			var valid func(emulator.HookType, debugger.Context) bool
			var callback debugger.MemoryCallback
			var data any
			if handler, ok := hook.(*memHandler); ok {
				valid = handler.valid
				callback = handler.callback
				data = handler.data
			} else if handler, ok := hook.(*hookHandler[debugger.MemoryCallback]); ok {
				valid = handler.valid
				callback = handler.callback
				data = handler.data
			}
			if valid(typ, ctx) {
				result = callback(ctx, typ, addr, size, value, data)
				if result == debugger.HookResult_Done {
					break
				}
			}
		}
		if result == debugger.HookResult_Next {
			task.CancelCause(debugger.NewInvalidMemoryException(ctx, typ, addr, size, value))
		}
	})
	return true
}

func (h *hookHandler[T]) Close() error {
	for i := len(h.releases) - 1; i >= 0; i-- {
		h.releases[i]()
	}
	h.releases = nil
	return nil
}

func (h *hookHandler[T]) Type() emulator.HookType {
	return h.typ
}

func (h *hookHandler[T]) valid(typ emulator.HookType, ctx debugger.Context) bool {
	if h.typ&typ == 0 {
		return false
	} else if h.begin > h.end {
		return true
	}
	pc, err := ctx.RegRead(ctx.PC())
	if err != nil {
		return false
	}
	return pc >= h.begin && pc < h.end
}

func (h *codeHandler) handleCode(addr, size uint64, data any) {
	data.(Debugger).syncTask(func(task debugger.Task) {
		h.callback(task.Context(), addr, size, h.data)
	})
}

func (h *memHandler) handleMemory(typ emulator.HookType, addr, size, value uint64, data any) bool {
	data.(Debugger).syncTask(func(task debugger.Task) {
		h.callback(task.Context(), typ, addr, size, value, h.data)
	})
	return true
}

func (h *controlHandler) Close() error {
	for i := len(h.releases) - 1; i >= 0; i-- {
		h.releases[i]()
	}
	return nil
}

func (h *controlHandler) Addr() uint64 {
	return h.addr[0]
}

func (h *controlHandler) handleControl(ctx debugger.Context, intno uint64, data any) debugger.HookResult {
	h.callback(ctx, data)
	return debugger.HookResult_Done
}

func (dbg *Dbg) AddHook(typ emulator.HookType, callback any, data any, begin, end uint64) (debugger.HookHandler, error) {
	return dbg.hookManger.addHook(dbg.impl, typ, callback, data, begin, end)
}

func (dbg *Dbg) AddControl(callback debugger.ControlCallback, data any) (debugger.ControlHandler, error) {
	return dbg.hookManger.addControl(dbg.impl, callback, data)
}
