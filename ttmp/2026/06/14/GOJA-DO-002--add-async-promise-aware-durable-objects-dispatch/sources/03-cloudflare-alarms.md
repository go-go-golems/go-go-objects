---
Title: Cloudflare Durable Objects Alarms API
Ticket: GOJA-DO-002
Status: active
Topics:
    - goja
    - durable-objects
    - actor-runtime
DocType: source
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Source/evidence material for GOJA-DO-002 async Durable Objects dispatch design.
LastUpdated: 2026-06-14T18:00:00Z
WhatFor: Use as source evidence for Promise-aware Durable Objects dispatch design.
WhenToUse: Read when validating claims in the async dispatch design guide.
---

## Background

Durable Objects alarms allow you to schedule the Durable Object to be woken up at a time in the 
future. When the alarm's scheduled time comes, the `alarm()` handler method will be called. Alarms 
are modified using the Storage API, and alarm operations follow the same rules as other storage 
operations.

Notably:

- Each Durable Object is able to schedule a single alarm at a time by calling `setAlarm()`.
- Alarms have guaranteed at-least-once execution and are retried automatically when the `alarm()` 
handler throws.
- Retries are performed using exponential backoff starting at a 2 second delay from the first 
failure with up to 6 retries allowed.

Alarms can be used to build distributed primitives, like queues or batching of work atop Durable 
Objects. Alarms also provide a mechanism to guarantee that operations within a Durable Object will 
complete without relying on incoming requests to keep the Durable Object alive. For a complete 
example, refer to [Use the Alarms 
API](https://developers.cloudflare.com/durable-objects/examples/alarms-api/).

## Scheduling multiple events with a single alarm

Although each Durable Object can only have one alarm set at a time, you can manage many scheduled 
and recurring events by storing your event schedule in storage and having the `alarm()` handler 
process due events, then reschedule itself for the next one.

```js
import { DurableObject } from "cloudflare:workers";

export class AgentServer extends DurableObject {
  // Schedule a one-time or recurring event
  async scheduleEvent(id, runAt, repeatMs = null) {
    await this.ctx.storage.put(\`event:${id}\`, { id, runAt, repeatMs });
    const currentAlarm = await this.ctx.storage.getAlarm();
    if (!currentAlarm || runAt < currentAlarm) {
      await this.ctx.storage.setAlarm(runAt);
    }
  }

  async alarm() {
    const now = Date.now();
    const events = await this.ctx.storage.list({ prefix: "event:" });
    let nextAlarm = null;

    for (const [key, event] of events) {
      if (event.runAt <= now) {
        await this.processEvent(event);
        if (event.repeatMs) {
          event.runAt = now + event.repeatMs;
          await this.ctx.storage.put(key, event);
        } else {
          await this.ctx.storage.delete(key);
        }
      }
      // Track the next event time
      if (event.runAt > now && (!nextAlarm || event.runAt < nextAlarm)) {
        nextAlarm = event.runAt;
      }
    }

    if (nextAlarm) await this.ctx.storage.setAlarm(nextAlarm);
  }

  async processEvent(event) {
    // Your event handling logic here
  }
}
```

## Storage methods

### getAlarm

- `getAlarm()`: `  number | null  `
	- If there is an alarm set, then return the currently set alarm time as the number of 
milliseconds elapsed since the UNIX epoch. Otherwise, return `null`.
		- If `getAlarm` is called while an 
[`alarm`](https://developers.cloudflare.com/durable-objects/api/alarms/#alarm) is already running, 
it returns `null` unless `setAlarm` has also been called since the alarm handler started running.

### setAlarm

- ``  setAlarm(scheduledTimeMs `  number  `)  ``: `  void  `
	- Set the time for the alarm to run. Specify the time as the number of milliseconds elapsed 
since the UNIX epoch.
		- If you call `setAlarm` when there is already one scheduled, it will override the 
existing alarm.

### deleteAlarm

- `deleteAlarm()`: `  void  `
	- Unset the alarm if there is a currently set alarm.
		- Calling `deleteAlarm()` inside the `alarm()` handler may prevent retries on a 
best-effort basis, but is not guaranteed.

## Handler methods

### alarm

- ``alarm(alarmInfo `  Object  `)``: `  void  `
	- Called by the system when a scheduled alarm time is reached.
		- The optional parameter `alarmInfo` object has two properties:
		- `retryCount` `  number  `: The number of times this alarm event has been retried.
				- `isRetry` `  boolean  `: A boolean value to indicate if the alarm 
has been retried. This value is `true` if this alarm event is a retry.
		- Only one instance of `alarm()` will ever run at a given time per Durable Object 
instance.
		- The `alarm()` handler has guaranteed at-least-once execution and will be retried 
upon failure using exponential backoff, starting at 2 second delays for up to 6 retries. This only 
applies to the most recent `setAlarm()` call. Retries will be performed if the method fails with an 
uncaught exception.
		- This method can be `async`.

## Example

This example shows how to both set alarms with the `setAlarm(timestamp)` method and handle alarms 
with the `alarm()` handler within your Durable Object.

- The `alarm()` handler will be called once every time an alarm fires.
- If an unexpected error terminates the Durable Object, the `alarm()` handler may be 
re-instantiated on another machine.
- Following a short delay, the `alarm()` handler will run from the beginning on the other machine.

- [JavaScript](#tab-panel-8023)
- [Python](#tab-panel-8024)

```js
import { DurableObject } from "cloudflare:workers";

export default {
  async fetch(request, env) {
    return await env.ALARM_EXAMPLE.getByName("foo").fetch(request);
  },
};

const SECONDS = 1000;

export class AlarmExample extends DurableObject {
  constructor(ctx, env) {
    super(ctx, env);
    this.storage = ctx.storage;
  }
  async fetch(request) {
    // If there is no alarm currently set, set one for 10 seconds from now
    let currentAlarm = await this.storage.getAlarm();
    if (currentAlarm == null) {
      this.storage.setAlarm(Date.now() + 10 * SECONDS);
    }
  }
  async alarm() {
    // The alarm handler will be invoked whenever an alarm fires.
    // You can use this to do work, read from the Storage API, make HTTP calls
    // and set future alarms to run using this.storage.setAlarm() from within this handler.
  }
}
```

The following example shows how to use the `alarmInfo` property to identify if the alarm event has 
been attempted before.

- [JavaScript](#tab-panel-8025)
- [Python](#tab-panel-8026)

```js
class MyDurableObject extends DurableObject {
  async alarm(alarmInfo) {
    if (alarmInfo?.retryCount != 0) {
      console.log(
        "This alarm event has been attempted ${alarmInfo?.retryCount} times before.",
      );
    }
  }
}
```

## Related resources

- Understand how to [use the Alarms 
API](https://developers.cloudflare.com/durable-objects/examples/alarms-api/) in an end-to-end 
example.
- Read the [Durable Objects alarms announcement blog post 
↗](https://blog.cloudflare.com/durable-objects-alarms/).
- Review the [Storage 
API](https://developers.cloudflare.com/durable-objects/api/sqlite-storage-api/) documentation for 
Durable Objects.
