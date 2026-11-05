1. **Add `EnvelopeValidation` Middleware (`internal/app/worker/router.go`)**
   - Import necessary packages like `github.com/ThreeDotsLabs/watermill/message`.
   - Create a `ValidationMiddleware` that:
     1. Unmarshals `msg.Payload` into a struct to extract `event_id` and `occurred_at`.
     2. If unmarshalling fails or `event_id` / `occurred_at` are empty, log the validation error, along with `event_id` (if parsed), the topic (using `message.SubscribeTopicFromCtx(msg.Context())`), and `correlation_id` (from headers or parsed envelope). Ensure we DO NOT log the payload (CWE-532).
     3. Return an error to stop processing this message.

2. **Add Custom Poison Queue Middleware (`internal/app/worker/router.go`)**
   - Standard Watermill `PoisonQueue` publishes to a fixed topic, but the requirement says `<topic>.dlq`.
   - We will write a custom PoisonQueue middleware `PoisonQueue(pub PubSub)` that:
     1. Calls the next handler `events, err := h(msg)`.
     2. If `err != nil`, get the current topic `topic := message.SubscribeTopicFromCtx(msg.Context())`.
     3. Publish a copy of `msg` to `topic + ".dlq"`.
     4. Return `nil, nil` so the original message is acked and not re-processed by the broker.

3. **Wire Middlewares in `CreateRouter` (`internal/app/worker/router.go`)**
   - Instantiate `middleware.Retry` with:
     - `MaxRetries: 3`
     - `InitialInterval: 50 * time.Millisecond`
     - `MaxInterval: 1 * time.Second`
     - `Multiplier: 2.0`
   - Apply middlewares to the router in the correct order:
     1. `router.AddMiddleware(PoisonQueue(ps))` (catches errors, publishes to `.dlq`).
     2. `router.AddMiddleware(ValidationMiddleware)` (rejects invalid envelopes immediately, straight to PoisonQueue).
     3. `router.AddMiddleware(retry.Middleware)` (retries handlers that fail).
     *(Wait, if Validation is before Retry, it returns an error to PoisonQueue immediately. If Validation passes, it calls Retry. Retry calls the Handler. If Handler fails, Retry retries it. If Retry exhausts retries, it returns an error, which Validation will pass through to PoisonQueue. Yes, this is correct).*

4. **Verify Tests in `worker_test.go`**
   - Add a test checking the `.dlq` behavior. Send malformed, missing-field, and handler-error messages and assert they end up in the `.dlq` channel.

5. **Pre-commit Steps**
   - Complete pre-commit steps to ensure proper testing, verification, review, and reflection are done.

6. **Submit PR**
