package cmd

var delegatedProcessProtocol = virtualProtocol{
	ID:          "virtual-nn-delegated-process",
	Title:       "Protocol: review delegated process through nn transcript",
	AppliesWhen: "before judging, accepting, rejecting, steering, continuing, or extracting substantive delegated work or conclusions when retained transcript evidence is available and the decision depends on what the agent actually established",
	Body: "When evaluating the process followed by an agent, load `nn-transcript` and inspect that agent’s attributable retained evidence directly. Use assignment context, bounded recent events, explicit failures, and parent-side handoffs as required by the question.\n\n" +
		"Do not infer process quality, current activity, success, drift, or a need to steer from lifecycle status, token counts, summaries, or missing returns alone.\n\n" +
		"Before steering, inspect the latest relevant sequence and distinguish:\n" +
		"- legitimate investigation from unresolved repetition,\n" +
		"- planned RED tests from unexpected failures,\n" +
		"- agent reports from inspected supporting results,\n" +
		"- producer completion from successful parent handoff.\n\n" +
		"Keep the read proportional: use bounded native transcript operations first and expand only when the conclusion depends on omitted evidence.\n\n" +
		"A branch diff or final summary may identify what to inspect, but neither substitutes for transcript evidence when adopting the predecessor’s reasoning or claimed RED/GREEN boundaries.\n",
}
