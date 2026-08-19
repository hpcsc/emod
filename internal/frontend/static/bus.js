export function createBus() {
  const listeners = {};

  return {
    on(event, fn) {
      (listeners[event] ||= []).push(fn);
    },
    off(event, fn) {
      const fns = listeners[event];
      if (fns) {
        listeners[event] = fns.filter(f => f !== fn);
      }
    },
    emit(event, data) {
      (listeners[event] || []).forEach(fn => fn(data));
    },
  };
}

// Every viewer module talks over this one shared bus.
export const bus = createBus();
