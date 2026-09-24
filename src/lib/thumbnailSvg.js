/**
 * Dynamic fallback video thumbnail generator for Aurahub.
 * Generates rich, branded 16:9 SVGs with deterministic gradients based on seed/title.
 */

function fnv1a(str) {
  let hash = 2166136261;
  const s = String(str || "aurahub");
  for (let i = 0; i < s.length; i++) {
    hash ^= s.charCodeAt(i);
    hash += (hash << 1) + (hash << 4) + (hash << 7) + (hash << 8) + (hash << 24);
  }
  return hash >>> 0;
}

const PALETTES = [
  { from: "#4338ca", to: "#6d28d9", accent: "#a78bfa", name: "Indigo Purple" },
  { from: "#0f172a", to: "#1e1b4b", accent: "#818cf8", name: "Midnight Nebula" },
  { from: "#065f46", to: "#0f766e", accent: "#34d399", name: "Emerald Teal" },
  { from: "#be123c", to: "#7e22ce", accent: "#fb7185", name: "Crimson Violet" },
  { from: "#0369a1", to: "#1d4ed8", accent: "#38bdf8", name: "Oceanic Blue" },
  { from: "#b45309", to: "#b91c1c", accent: "#fbbf24", name: "Sunset Amber" },
  { from: "#18181b", to: "#27272a", accent: "#a5b4fc", name: "Carbon Slate" },
  { from: "#581c87", to: "#831843", accent: "#f472b6", name: "Neon Cyber" },
];

function escapeXml(unsafe) {
  return String(unsafe || "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&apos;");
}

/**
 * Generates a 16:9 branded SVG thumbnail string.
 * @param {object} options
 * @param {string} options.seed - Identifier for deterministic color (video ID or title)
 * @param {string} [options.title] - Video title to render
 * @param {string} [options.category] - Video category (e.g. Tech, Gaming)
 * @returns {string} Complete SVG XML markup
 */
export function generateThumbnailSvg({ seed, title, category }) {
  const cleanSeed = seed || title || "aurahub";
  const hash = fnv1a(cleanSeed);
  const palette = PALETTES[hash % PALETTES.length];

  const rawTitle = title || "Aurahub Video";
  const cleanCat = escapeXml(category || "Video").toUpperCase();

  // Split title into at most 2 display lines
  const words = rawTitle.split(/\s+/).filter(Boolean);
  let line1 = "";
  let line2 = "";

  for (const w of words) {
    if ((line1 + " " + w).length <= 26 && !line2) {
      line1 += (line1 ? " " : "") + w;
    } else if ((line2 + " " + w).length <= 28) {
      line2 += (line2 ? " " : "") + w;
    }
  }

  if (words.length > 0 && line2.length > 25) {
    line2 = line2.slice(0, 24) + "...";
  }

  const escapedLine1 = escapeXml(line1 || rawTitle.slice(0, 26));
  const escapedLine2 = escapeXml(line2);

  const catBadgeWidth = Math.max(80, cleanCat.length * 11 + 28);

  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1280 720" width="100%" height="100%">
  <defs>
    <linearGradient id="bg" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="${palette.from}"/>
      <stop offset="100%" stop-color="${palette.to}"/>
    </linearGradient>
    <radialGradient id="glow" cx="80%" cy="25%" r="65%">
      <stop offset="0%" stop-color="${palette.accent}" stop-opacity="0.32"/>
      <stop offset="100%" stop-color="${palette.from}" stop-opacity="0"/>
    </radialGradient>
    <pattern id="grid" width="40" height="40" patternUnits="userSpaceOnUse">
      <path d="M 40 0 L 0 0 0 40" fill="none" stroke="rgba(255,255,255,0.04)" stroke-width="1"/>
    </pattern>
    <filter id="shadow" x="-20%" y="-20%" width="140%" height="140%">
      <feDropShadow dx="0" dy="8" stdDeviation="12" flood-opacity="0.4"/>
    </filter>
  </defs>

  <!-- Background Base -->
  <rect width="1280" height="720" fill="url(#bg)"/>
  <rect width="1280" height="720" fill="url(#glow)"/>
  <rect width="1280" height="720" fill="url(#grid)"/>

  <!-- Frosted Center Play Icon -->
  <g transform="translate(940, 320)" filter="url(#shadow)">
    <circle cx="40" cy="40" r="68" fill="rgba(255,255,255,0.12)" stroke="rgba(255,255,255,0.28)" stroke-width="2"/>
    <polygon points="33,22 63,40 33,58" fill="#ffffff" opacity="0.95"/>
  </g>

  <!-- Top-Left Aurahub Brand Tag -->
  <g transform="translate(80, 80)">
    <rect width="134" height="38" rx="19" fill="rgba(0,0,0,0.35)" stroke="rgba(255,255,255,0.18)" stroke-width="1"/>
    <circle cx="24" cy="19" r="6" fill="${palette.accent}"/>
    <text x="38" y="24" font-family="-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif" font-size="14" font-weight="800" fill="#ffffff" letter-spacing="2">AURAHUB</text>
  </g>

  <!-- Top Category Tag -->
  <g transform="translate(234, 80)">
    <rect width="${catBadgeWidth}" height="38" rx="19" fill="rgba(255,255,255,0.15)" stroke="rgba(255,255,255,0.28)" stroke-width="1"/>
    <text x="14" y="24" font-family="-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif" font-size="13" font-weight="700" fill="#ffffff" letter-spacing="1.5">${cleanCat}</text>
  </g>

  <!-- Title & Accent Section -->
  <rect x="80" y="470" width="60" height="6" rx="3" fill="${palette.accent}"/>

  <text x="80" y="550" font-family="-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif" font-size="52" font-weight="800" fill="#ffffff" filter="url(#shadow)">
    ${escapedLine1}
  </text>
  ${
    escapedLine2
      ? `<text x="80" y="618" font-family="-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif" font-size="44" font-weight="700" fill="rgba(255,255,255,0.88)" filter="url(#shadow)">${escapedLine2}</text>`
      : ""
  }
</svg>`;
}

/**
 * Returns a fallback thumbnail URL for a video.
 * @param {string} videoId
 * @param {string} [title]
 * @param {string} [category]
 * @returns {string} URL string
 */
export function getFallbackThumbnailUrl(videoId, title, category) {
  const seed = encodeURIComponent(videoId || "default");
  const params = new URLSearchParams();
  if (title) params.set("title", title.slice(0, 80));
  if (category) params.set("category", category);

  const query = params.toString();
  return `/api/thumbnail/${seed}${query ? `?${query}` : ""}`;
}
