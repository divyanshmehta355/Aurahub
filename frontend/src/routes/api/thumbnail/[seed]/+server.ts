import { generateThumbnailSvg } from '#lib/thumbnailSvg';

export async function GET({ params, url }) {
	const { seed } = params;
	const title = url.searchParams.get('title') || '';
	const category = url.searchParams.get('category') || '';

	const svgString = generateThumbnailSvg({
		seed: decodeURIComponent(seed),
		title: decodeURIComponent(title),
		category: decodeURIComponent(category)
	});

	return new Response(svgString, {
		headers: {
			'Content-Type': 'image/svg+xml',
			'Cache-Control': 'public, max-age=31536000, immutable'
		}
	});
}
