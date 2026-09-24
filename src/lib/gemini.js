/**
 * Gemini embedding generation and vector similarity utilities.
 * Uses official Generative Language API with supported embedding models.
 */

const EMBEDDING_MODELS = [
  "gemini-embedding-001",
  "gemini-embedding-2",
];

/**
 * Generate a 768-dimensional text embedding using Gemini.
 * @param {string} text - Input text content to embed
 * @returns {Promise<number[]|null>} - 768-dimensional vector
 */
export async function generateEmbedding(text) {
  if (!text || typeof text !== "string" || !text.trim()) return null;

  const apiKey = process.env.GEMINI_API_KEY;
  if (!apiKey) {
    return null;
  }

  const cleanText = text.slice(0, 8000);

  for (const model of EMBEDDING_MODELS) {
    try {
      const response = await fetch(
        `https://generativelanguage.googleapis.com/v1beta/models/${model}:embedContent?key=${apiKey}`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            content: { parts: [{ text: cleanText }] },
            outputDimensionality: 768,
          }),
        }
      );

      if (!response.ok) {
        continue;
      }

      const data = await response.json();
      const values = data?.embedding?.values;
      if (Array.isArray(values) && values.length > 0) {
        return values;
      }
    } catch (error) {
      // Try next model
    }
  }

  console.error("Gemini embedding generation error: all models failed or rejected request");
  return null;
}

/**
 * Creates formatted metadata text and generates an embedding for a video document.
 * @param {object} video - Video document or metadata object
 * @returns {Promise<number[]|null>}
 */
export async function generateVideoEmbedding(video) {
  if (!video) return null;

  const parts = [];
  if (video.title) parts.push(`Title: ${video.title}`);
  if (video.category) parts.push(`Category: ${video.category}`);
  if (video.tags && video.tags.length > 0) {
    const tagList = Array.isArray(video.tags) ? video.tags.join(", ") : video.tags;
    parts.push(`Tags: ${tagList}`);
  }
  if (video.description) parts.push(`Description: ${video.description.slice(0, 1000)}`);

  const fullText = parts.join("\n");
  return await generateEmbedding(fullText);
}

/**
 * Computes cosine similarity between two vector arrays of equal length.
 * @param {number[]} vecA
 * @param {number[]} vecB
 * @returns {number} - Similarity score between -1.0 and 1.0 (typically 0.0 - 1.0)
 */
export function cosineSimilarity(vecA, vecB) {
  if (!Array.isArray(vecA) || !Array.isArray(vecB) || vecA.length !== vecB.length) {
    return 0;
  }

  let dotProduct = 0;
  let normA = 0;
  let normB = 0;

  for (let i = 0; i < vecA.length; i++) {
    dotProduct += vecA[i] * vecB[i];
    normA += vecA[i] * vecA[i];
    normB += vecB[i] * vecB[i];
  }

  if (normA === 0 || normB === 0) return 0;
  return dotProduct / (Math.sqrt(normA) * Math.sqrt(normB));
}

/**
 * Computes the weighted average vector (user taste profile) from interacted videos.
 * @param {Array<{embedding: number[], weight: number}>} interactions
 * @returns {number[]|null} - The user's taste vector
 */
export function computeUserTasteVector(interactions) {
  if (!Array.isArray(interactions) || interactions.length === 0) return null;

  const valid = interactions.filter(
    (item) => Array.isArray(item.embedding) && item.embedding.length > 0
  );
  if (valid.length === 0) return null;

  const dim = valid[0].embedding.length;
  const result = new Array(dim).fill(0);
  let totalWeight = 0;

  for (const item of valid) {
    const weight = item.weight || 1;
    totalWeight += weight;
    for (let i = 0; i < dim; i++) {
      result[i] += item.embedding[i] * weight;
    }
  }

  if (totalWeight === 0) return null;

  for (let i = 0; i < dim; i++) {
    result[i] /= totalWeight;
  }

  return result;
}

/**
 * Uses Gemini Flash to detect typos and return a corrected query string.
 * Returns null if the query is already correct or if check fails.
 * @param {string} query
 * @returns {Promise<string|null>}
 */
export async function correctSearchQuery(query) {
  if (!query || typeof query !== "string" || query.trim().length < 3) {
    return null;
  }

  const apiKey = process.env.GEMINI_API_KEY;
  if (!apiKey) return null;

  const cleanQuery = query.trim();

  try {
    const res = await fetch(
      `https://generativelanguage.googleapis.com/v1beta/models/gemini-3.6-flash:generateContent?key=${apiKey}`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          contents: [
            {
              parts: [
                {
                  text: `You are a search query spell-checker for a video platform.
If the query has typos or misspellings, output ONLY the corrected query.
If the query is already correct, output ONLY the word "CORRECT".
Do not output quotes, explanations, or punctuation.

Query: "${cleanQuery}"`,
                },
              ],
            },
          ],
          generationConfig: {
            maxOutputTokens: 20,
            temperature: 0.1,
          },
        }),
      }
    );

    if (!res.ok) return null;

    const data = await res.json();
    const text = data?.candidates?.[0]?.content?.parts?.[0]?.text?.trim();

    if (
      text &&
      text !== "CORRECT" &&
      text.toLowerCase() !== cleanQuery.toLowerCase() &&
      !text.toLowerCase().includes("correct")
    ) {
      // Clean any enclosing quotes or periods
      return text.replace(/^["']|["']$/g, "").replace(/\.$/, "").trim();
    }

    return null;
  } catch (err) {
    return null;
  }
}

