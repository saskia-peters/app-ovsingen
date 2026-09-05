// Package ports declares the inbound/outbound port interfaces of the User
// Directory & Auth hexagon (AD-1/AD-2). Across the module boundary only these
// interfaces are visible: the single auth port (`Service`) that Tool
// Maintenance and Admin consume, and the repository ports driven by the
// User hexagon. Concrete interfaces land with stories 1.3-1.10.
package ports