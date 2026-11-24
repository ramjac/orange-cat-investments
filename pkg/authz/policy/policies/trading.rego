package oci.authz.trading

import rego.v1

default allow := false

blocked_account_statuses := {"closed", "suspended", "frozen", "erased"}

max_order_notional_usd := object.get(input.limits, "max_order_notional_usd", 250000)

allow if {
	count(deny_reasons) == 0
}

deny_reasons contains msg if {
	input.account.status in blocked_account_statuses
	msg := sprintf("trading prohibited: account status is %q", [input.account.status])
}

deny_reasons contains msg if {
	input.feline.emergency_medical_hold == true
	not input.actor.role in {"risk_override", "admin"}
	msg := sprintf("algorithmic trading halted: emergency medical hold active for feline executive %s", [input.feline.feline_id])
}

deny_reasons contains msg if {
	input.order.quantity <= 0
	msg := "order quantity must be strictly positive"
}

deny_reasons contains msg if {
	not input.order.side in {"buy", "sell"}
	msg := sprintf("unsupported order side %q: must be 'buy' or 'sell'", [input.order.side])
}

deny_reasons contains msg if {
	input.order.estimated_notional_usd > max_order_notional_usd
	msg := sprintf("order notional $%.2f exceeds single-order policy limit of $%.2f", [input.order.estimated_notional_usd, max_order_notional_usd])
}

deny_reasons contains msg if {
	input.order.side == "buy"
	input.account.buying_power_usd > 0
	input.order.estimated_notional_usd > input.account.buying_power_usd
	msg := sprintf("insufficient buying power: estimated order notional $%.2f exceeds available $%.2f", [input.order.estimated_notional_usd, input.account.buying_power_usd])
}
