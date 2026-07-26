package realm

// Artefact identifies which realm artefact a provisioning result concerns.
type Artefact string

// The four artefacts a realm holds.
const (
	ArtefactStream      Artefact = "stream"
	ArtefactNotify      Artefact = "notify_stream"
	ArtefactObjectStore Artefact = "object_store"
	ArtefactPersonas    Artefact = "personas"
)

// Outcome is the provisioning result for a single artefact.
type Outcome string

const (
	// OutcomeCreated means the artefact was missing and has now been created.
	OutcomeCreated Outcome = "created"
	// OutcomeConformant means the artefact already existed and matches the mandated shape.
	OutcomeConformant Outcome = "conformant"
	// OutcomeNonconformant means the artefact already existed but drifts from the
	// mandated shape. It is reported, never mutated.
	OutcomeNonconformant Outcome = "nonconformant"
	// OutcomeUpdated means the artefact held the recognised legacy shape and was
	// deliberately converged to the mandate (the one exception to report-only).
	OutcomeUpdated Outcome = "updated"
)

// ArtefactResult is the provisioning outcome for one artefact. Nonconformities is
// populated only when Outcome is [OutcomeNonconformant]; each entry names one drift.
// MaxBytes is the artefact's byte roof (0 = unlimited): the budget applied when the
// outcome is [OutcomeCreated], the roof as found on the server otherwise.
type ArtefactResult struct {
	Artefact        Artefact
	Outcome         Outcome
	Nonconformities []string
	MaxBytes        int64
}

// ProvisionReport is the structured result of a provisioning run.
type ProvisionReport struct {
	Results []ArtefactResult
}

// Conformant reports whether every artefact is present and conformant — that is,
// nothing was reported as nonconformant.
func (r *ProvisionReport) Conformant() bool {
	for _, res := range r.Results {
		if res.Outcome == OutcomeNonconformant {
			return false
		}
	}
	return true
}

// result builds an ArtefactResult from a (possibly empty) list of nonconformities.
func result(a Artefact, nonconformities []string) ArtefactResult {
	if len(nonconformities) == 0 {
		return ArtefactResult{Artefact: a, Outcome: OutcomeConformant}
	}
	return ArtefactResult{Artefact: a, Outcome: OutcomeNonconformant, Nonconformities: nonconformities}
}
