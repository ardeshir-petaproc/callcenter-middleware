package server

import (
	"net/http"

	"callcenter-middleware/internal/domain"
)

// jobTemplate is a ready-made job description a manager can start from.
type jobTemplate struct {
	domain.JobProfile
	TitleFA string `json:"title_fa"`
}

var jobTemplates = []jobTemplate{
	{TitleFA: "کارشناس فروش", JobProfile: domain.JobProfile{Title: "Sales representative",
		Description:          "Handles pre-sales questions, quotes and plan upgrades for new and existing customers.",
		Responsibilities:     []string{"Answer product and pricing questions", "Prepare quotes", "Process upgrades"},
		RequestTypes:         []string{"sales", "pricing", "upgrade"},
		AuthorityLimits:      "May offer discounts up to 10%. Larger discounts need the sales manager.",
		EscalationConditions: "Discount above 10%, enterprise contracts, legal questions."}},
	{TitleFA: "کارشناس پشتیبانی سطح ۱", JobProfile: domain.JobProfile{Title: "Support agent (L1)",
		Description:          "First line support for account access and common product issues.",
		Responsibilities:     []string{"Reset passwords and unlock accounts", "Troubleshoot common issues", "Open tickets"},
		RequestTypes:         []string{"login_issue", "technical_support", "how_to"},
		AuthorityLimits:      "Cannot change billing data or issue refunds.",
		EscalationConditions: "Outages, data loss, security incidents, issues unresolved after 15 minutes."}},
	{TitleFA: "کارشناس پشتیبانی فنی سطح ۲", JobProfile: domain.JobProfile{Title: "Technical support (L2)",
		Description:          "Investigates complex technical problems escalated from L1.",
		Responsibilities:     []string{"Diagnose integrations and errors", "Coordinate with engineering"},
		RequestTypes:         []string{"technical_support", "integration", "bug_report"},
		AuthorityLimits:      "May grant temporary service credits up to one month.",
		EscalationConditions: "Confirmed bugs affecting many customers go to engineering on-call."}},
	{TitleFA: "کارشناس حسابداری و صورتحساب", JobProfile: domain.JobProfile{Title: "Billing specialist",
		Description:          "Resolves invoices, payments and refund requests.",
		Responsibilities:     []string{"Explain invoices", "Process refunds", "Update payment methods"},
		RequestTypes:         []string{"billing", "refund", "payment"},
		AuthorityLimits:      "Refunds up to 5,000,000 IRR (or equivalent). Larger refunds need the finance manager.",
		EscalationConditions: "Chargebacks, disputed contracts, refunds above the limit."}},
	{TitleFA: "کارشناس رسیدگی به شکایات", JobProfile: domain.JobProfile{Title: "Complaints officer",
		Description:          "Owns formal complaints until resolution.",
		Responsibilities:     []string{"Record complaints", "Investigate and respond within SLA", "Propose remedies"},
		RequestTypes:         []string{"complaint"},
		AuthorityLimits:      "May offer goodwill credits; compensation above policy needs the customer affairs manager.",
		EscalationConditions: "Legal threats, media exposure, safety issues."}},
	{TitleFA: "سرپرست مرکز تماس", JobProfile: domain.JobProfile{Title: "Contact center supervisor",
		Description:          "Supervises agents, handles escalations and monitors quality.",
		Responsibilities:     []string{"Handle escalated calls", "Monitor queues and quality", "Coach agents"},
		RequestTypes:         []string{"escalation", "complaint"},
		AuthorityLimits:      "May approve exceptions within company policy.",
		EscalationConditions: "Anything outside policy goes to the department manager."}},
}

func (s *Server) jobTemplates(w http.ResponseWriter, r *http.Request) error {
	writeJSON(w, http.StatusOK, jobTemplates)
	return nil
}
