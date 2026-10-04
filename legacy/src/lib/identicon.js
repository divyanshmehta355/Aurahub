/**
 * Pure JavaScript GitHub-style 5x5 Identicon generator.
 * Zero external dependencies.
 */

function fnv1a(str) {
  let hash = 2166136261;
  for (let i = 0; i < str.length; i++) {
    hash ^= str.charCodeAt(i);
    hash += (hash << 1) + (hash << 4) + (hash << 7) + (hash << 8) + (hash << 24);
  }
  return hash >>> 0;
}

/**
 * Generates an SVG string containing a GitHub-style 5x5 symmetric identicon.
 * @param {string} seed - The identifier (username, user ID, email)
 * @returns {string} - Complete SVG markup
 */
export function generateIdenticonSvg(seed) {
  const cleanSeed = String(seed || "user").trim().toLowerCase();

  // Create 16 deterministic pseudo-random bytes from the seed
  const bytes = [];
  const h1 = fnv1a(cleanSeed);
  const h2 = fnv1a(cleanSeed + "_prime");
  const h3 = fnv1a(cleanSeed + "_seed");
  const h4 = fnv1a(cleanSeed + "_identicon");

  for (let i = 0; i < 4; i++) bytes.push((h1 >> (i * 8)) & 0xff);
  for (let i = 0; i < 4; i++) bytes.push((h2 >> (i * 8)) & 0xff);
  for (let i = 0; i < 4; i++) bytes.push((h3 >> (i * 8)) & 0xff);
  for (let i = 0; i < 4; i++) bytes.push((h4 >> (i * 8)) & 0xff);

  // Pick deterministic HSL color
  const hue = ((bytes[0] << 8) | bytes[1]) % 360;
  const saturation = 65 + (bytes[2] % 20); // 65% - 85%
  const lightness = 44 + (bytes[3] % 14);  // 44% - 58%
  const fgColor = `hsl(${hue}, ${saturation}%, ${lightness}%)`;
  const bgColor = "#f1f3f5";

  const cellSize = 70;
  const margin = 35;
  const size = margin * 2 + cellSize * 5; // 420px

  let rects = "";
  let filledCount = 0;

  // 5 rows x 3 columns (columns 3 and 4 mirror columns 1 and 0)
  for (let row = 0; row < 5; row++) {
    for (let col = 0; col < 3; col++) {
      const idx = row * 3 + col;
      const isFilled = (bytes[idx % bytes.length] + idx) % 2 === 0;

      if (isFilled) {
        filledCount++;
        const y = margin + row * cellSize;

        // Left / center column
        const x1 = margin + col * cellSize;
        rects += `<rect x="${x1}" y="${y}" width="${cellSize}" height="${cellSize}" fill="${fgColor}"/>`;

        // Mirrored column (col 1 -> col 3, col 0 -> col 4)
        if (col < 2) {
          const x2 = margin + (4 - col) * cellSize;
          rects += `<rect x="${x2}" y="${y}" width="${cellSize}" height="${cellSize}" fill="${fgColor}"/>`;
        }
      }
    }
  }

  // Ensure non-empty pattern
  if (filledCount === 0) {
    const yCenter = margin + 2 * cellSize;
    const xCenter = margin + 2 * cellSize;
    rects = `<rect x="${xCenter}" y="${yCenter}" width="${cellSize}" height="${cellSize}" fill="${fgColor}"/>`;
  }

  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${size} ${size}" width="100%" height="100%">
  <rect width="${size}" height="${size}" fill="${bgColor}" rx="60"/>
  ${rects}
</svg>`;
}

/**
 * Returns a valid avatar URL for a user, automatically falling back to an identicon.
 * @param {string} username - User's username
 * @param {string} [avatar] - Custom avatar URL if uploaded
 * @returns {string} - URL to avatar or dynamic SVG identicon
 */
export function getAvatarUrl(username, avatar) {
  if (avatar && typeof avatar === "string" && avatar.trim() !== "") {
    return avatar;
  }
  const clean = encodeURIComponent(username || "user");
  return `/api/avatar/${clean}`;
}
