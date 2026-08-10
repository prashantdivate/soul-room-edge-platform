# Upgrade Compatibility Policy

Agent upgrades must preserve device identity, local queue state, job execution
state, deployment state, and downstream device identities.

Breaking protocol changes require capability negotiation and a compatibility
window. State migrations must be versioned and recoverable.
