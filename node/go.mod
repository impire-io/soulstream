module github.com/impire-io/soulstream/node

go 1.26.2

// The node always builds from a repo checkout (it is not `go install @latest`-able):
// it must see the soulstream packages of the SAME change-set — the public mcpserver
// surface lands together with the node that first embeds it.
replace github.com/impire-io/soulstream => ../
