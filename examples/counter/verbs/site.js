const express = require("express");
const durableobjects = require("durableobjects");

__package__({ name: "durableobjects" });
__verb__("site", { name: "site", short: "Mount Durable Objects gateway", output: "text" });

function site() {
  const app = express.app();
  const gateway = durableobjects.gateway();

  app.get("/healthz", (_req, res) => res.send("ok"));
  app.mount("/rpc", gateway);
  app.mount("/fetch", gateway);
  return "durableobjects gateway mounted";
}

module.exports = { site };
