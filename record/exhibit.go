package record

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// ExhibitVersion is the only supported exhibit document version.
const ExhibitVersion = 1

// Exhibit is a portable, self-contained capture of one operation: the wire form
// verbatim (headers and payload, unknown Soulstream-* extras included) plus the two
// inputs verification needs — the realm and the canonical binding the signature was
// computed over. Because the bytes are verbatim, the author's signature keeps
// verifying wherever the document travels; because verification recomputes the
// canonical form from the stored realm and binding, a tampered exhibit cannot lie
// usefully — any alteration flips the verdict to failed.
//
// The subject is retained for human provenance display only; it plays no part in
// verification.
type Exhibit struct {
	Version int                 `json:"version"`
	Realm   string              `json:"realm"`
	Binding string              `json:"binding"`
	Subject string              `json:"subject,omitempty"`
	Headers map[string][]string `json:"headers"`
	Payload []byte              `json:"payload_b64"`
}

// Record reconstructs the captured operation via the standard wire parser.
func (e Exhibit) Record() (Record, error) {
	return Parse(e.Headers, e.Payload)
}

// Marshal serialises the exhibit to its stable document form. The same exhibit
// always marshals to the same bytes (struct field order and sorted map keys), so a
// re-marshalled round-trip is byte-identical.
func (e Exhibit) Marshal() ([]byte, error) {
	if err := e.check(); err != nil {
		return nil, err
	}
	data, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("record: marshal exhibit: %w", err)
	}
	return data, nil
}

// ParseExhibit strict-decodes an exhibit document: unknown fields are rejected
// loudly (a document claiming to be evidence gets no silent repair), the version
// must be supported, and the verification inputs must be present.
func ParseExhibit(data []byte) (Exhibit, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var e Exhibit
	if err := dec.Decode(&e); err != nil {
		return Exhibit{}, fmt.Errorf("record: parse exhibit: %w", err)
	}
	if dec.More() {
		return Exhibit{}, fmt.Errorf("record: parse exhibit: trailing data after document")
	}
	if err := e.check(); err != nil {
		return Exhibit{}, err
	}
	return e, nil
}

func (e Exhibit) check() error {
	if e.Version != ExhibitVersion {
		return fmt.Errorf("record: exhibit version %d unsupported (want %d)", e.Version, ExhibitVersion)
	}
	if e.Realm == "" {
		return fmt.Errorf("record: exhibit missing realm")
	}
	if e.Binding == "" {
		return fmt.Errorf("record: exhibit missing binding")
	}
	if len(e.Headers) == 0 {
		return fmt.Errorf("record: exhibit missing headers")
	}
	return nil
}
