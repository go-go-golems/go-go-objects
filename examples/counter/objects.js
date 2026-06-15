class Counter {
  constructor(state, env) {
    this.state = state;
    this.env = env;
  }

  async increment(by = 1) {
    await Promise.resolve();
    const current = this.state.storage.get("count") || 0;
    const next = current + by;
    this.state.storage.put("count", next);
    return next;
  }

  value() {
    return this.state.storage.get("count") || 0;
  }

  async fetch(req) {
    if (req.path === "/count") {
      return { status: 200, body: String(this.value()) };
    }
    return { status: 404, body: "not found" };
  }
}

exports.objects = { Counter };
