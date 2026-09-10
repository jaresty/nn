# ADR-0058: Compatibility-gated fastjson record decoding

Status: Evaluated; not adopted for production

Build a reusable decoder for the existing rawRecord union. Preserve raw message
and data bytes rather than reserializing fastjson values. Copy retained scalar
strings and payloads before parser/buffer reuse. Keep scanner framing, accepted
record ordinals and incomplete-line policy outside the adapter.

The common path requires standard JSON syntax and valid UTF-8. Known duplicate
keys, case-folded or escaped field names, Unicode escape sequences, incompatible
field types, parser errors, and records above a conservative 256 KiB fast-path
threshold use encoding/json. A fallback is not an omitted record. This threshold
bounds the fastjson parser's retained arena, not accepted input size.

Compare complete rawRecord values and admission against encoding/json, including
byte-identical RawMessage fields. Differential tests, retained-value tests and
fuzz seeds precede reader adoption. Benchmark complete decoding on the frozen
corpus and distinguish that result from earlier field-extraction measurements.
