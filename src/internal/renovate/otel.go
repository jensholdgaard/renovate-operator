package renovate

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel/trace"
)

// traceparentFromContext extracts the W3C traceparent string from the current
// span context. Returns "" when there is no valid span in the context.
func traceparentFromContext(ctx context.Context) string {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if !sc.IsValid() {
		return ""
	}
	return fmt.Sprintf("00-%s-%s-%s", sc.TraceID(), sc.SpanID(), sc.TraceFlags())
}

// otelWrapperEnabled returns true when the OTel wrapper should be injected
// into Renovate Job containers (forwarding enabled + active trace context).
func otelWrapperEnabled(traceparent string) bool {
	return os.Getenv("RENOVATE_FORWARD_OTEL") == "true" && traceparent != ""
}

// otelWrapperScript is a minimal Node.js script that creates a root span with
// a span link back to the operator's dispatch span (via TRACEPARENT), then
// spawns the original command. It resolves OTel packages from Renovate's own
// dependencies, so no additional packages need to be installed.
//
// Fault-tolerant: if OTel setup fails, the original command runs uninstrumented.
const otelWrapperScript = `'use strict';
const { spawn } = require('child_process');
const path = require('path');

const args = process.argv.slice(2);
if (args.length === 0) { process.exit(1); }

function run(cmd, cmdArgs) {
  return new Promise((resolve) => {
    const child = spawn(cmd, cmdArgs, { stdio: 'inherit' });
    child.on('error', () => resolve(1));
    child.on('close', (code) => resolve(code ?? 1));
  });
}

async function main() {
  const tp = process.env.TRACEPARENT;
  const endpoint = process.env.OTEL_EXPORTER_OTLP_ENDPOINT;
  if (!tp || !endpoint) {
    process.exit(await run(args[0], args.slice(1)));
  }

  let otel;
  try {
    const rdir = path.dirname(require.resolve('renovate/package.json'));
    const r = (p) => require(path.join(rdir, 'node_modules', p));

    const api = r('@opentelemetry/api');
    const { W3CTraceContextPropagator } = r('@opentelemetry/core');
    const { BasicTracerProvider, SimpleSpanProcessor } = r('@opentelemetry/sdk-trace-base');
    const { OTLPTraceExporter } = r('@opentelemetry/exporter-trace-otlp-http');

    const provider = new BasicTracerProvider();
    provider.addSpanProcessor(new SimpleSpanProcessor(new OTLPTraceExporter()));
    provider.register();

    const propagator = new W3CTraceContextPropagator();
    const ctx = propagator.extract(api.context.active(), { traceparent: tp }, {
      get: (c, k) => c[k],
      keys: (c) => Object.keys(c),
    });
    const parentSc = api.trace.getSpanContext(ctx);

    const span = provider.getTracer('renovate-otel-wrapper').startSpan('renovate.run', {
      kind: api.SpanKind.INTERNAL,
      links: parentSc ? [{ context: parentSc }] : [],
      attributes: { 'process.command_args': args.join(' ') },
    });

    otel = { api, span, provider };
  } catch (err) {
    console.error('otel-wrapper: setup failed, running without instrumentation:', err.message);
  }

  const code = await run(args[0], args.slice(1));

  if (otel) {
    try {
      if (code !== 0) {
        otel.span.setStatus({ code: otel.api.SpanStatusCode.ERROR, message: 'exit ' + code });
      }
      otel.span.setAttribute('process.exit_code', code);
      otel.span.end();
      await otel.provider.forceFlush();
      await otel.provider.shutdown();
    } catch (_) {}
  }

  process.exit(code);
}

main();
`

// wrapperPreamble returns a shell snippet that writes the OTel wrapper script
// to the scratch directory. The returned string should be prepended to shell commands.
func wrapperPreamble() string {
	return fmt.Sprintf("cat > \"${RENOVATE_BASE_DIR:-/tmp}/.otel-wrapper.js\" << 'OTEL_WRAPPER_EOF'\n%sOTEL_WRAPPER_EOF\n", otelWrapperScript)
}

// wrapCommand returns command and args that run the original command through
// the OTel wrapper. The wrapper is expected to have been written by wrapperPreamble.
func wrapCommand(originalCmd string, originalArgs ...string) string {
	result := fmt.Sprintf("node \"${RENOVATE_BASE_DIR:-/tmp}/.otel-wrapper.js\" %s", originalCmd)
	for _, a := range originalArgs {
		result += " " + a
	}
	return result
}
