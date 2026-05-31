const listeners = {};

export const bus = {
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
