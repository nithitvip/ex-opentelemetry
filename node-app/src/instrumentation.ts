import { NodeSDK } from '@opentelemetry/sdk-node';
import { ExpressInstrumentation } from '@opentelemetry/instrumentation-express';
import { HttpInstrumentation } from '@opentelemetry/instrumentation-http';

import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import { ConsoleSpanExporter, TraceIdRatioBasedSampler } from '@opentelemetry/sdk-trace-node';
import { ZipkinExporter }from '@opentelemetry/exporter-zipkin';


const sdk = new NodeSDK({
  traceExporter: new ConsoleSpanExporter(),
  // traceExporter: new OTLPTraceExporter({
  //   url: 'http://localhost:4318/v1/traces'
  // }),
  // traceExporter: new ZipkinExporter({
  //   url: 'http://localhost:9411/api/v2/spans'
  // }),
  instrumentations: [
    // getNodeAutoInstrumentations()
    new HttpInstrumentation(),
    new ExpressInstrumentation()
  ],
  // sampler: new TraceIdRatioBasedSampler(0.1),
  serviceName: "NodeDemoService"
  
});

sdk.start();