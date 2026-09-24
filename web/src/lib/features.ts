// Spaces stay hidden until the Teams gate exists (owner decision, 2026-09-23:
// "do not make it visible to everyone"). The server mirrors this with its
// --spaces flag (off by default), which refuses creating or joining a Space.
// When packages unlock shared spaces this becomes a per-user check.
export const SPACES_ENABLED = false;
