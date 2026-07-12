// Package realm connects to a Soulstream realm and provisions its two artefacts.
//
// A realm is one NATS account holding one SOULSTREAM stream (the op-log) and one
// soulstream-objects object store. [Connect] dials the server from a named NATS
// context and validates the realm name; [Client.Provision] brings the realm to its
// mandated shape.
//
// Provisioning is create-or-report: it creates only what is missing and reports the
// conformance of what already exists — it never modifies an existing artefact in
// place, because the history-destroying settings make in-place reconfiguration a
// one-way risk.
//
// This is the only package in the module that imports NATS.
package realm
