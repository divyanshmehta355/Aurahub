import { NextResponse } from 'next/server';
import { generateThumbnailSvg } from '@/lib/thumbnailSvg';
import dbConnect from '@/lib/dbConnect';
import Video from '@/models/Video';
import mongoose from 'mongoose';

export const dynamic = 'force-dynamic';

export async function GET(request, { params }) {
  try {
    const { seed } = await params;
    const decodedSeed = decodeURIComponent(seed || 'aurahub');

    const searchParams = request.nextUrl.searchParams;
    let title = searchParams.get('title') || '';
    let category = searchParams.get('category') || '';

    // If title was not passed in query params and seed is a valid Mongo ObjectId, fetch from DB
    if (!title && mongoose.Types.ObjectId.isValid(decodedSeed)) {
      try {
        await dbConnect();
        const video = await Video.findById(decodedSeed).select('title category').lean();
        if (video) {
          title = video.title || '';
          if (!category) {
            category = video.category || '';
          }
        }
      } catch (dbErr) {
        console.error('Error fetching video for fallback thumbnail:', dbErr);
      }
    }

    const svg = generateThumbnailSvg({
      seed: decodedSeed,
      title: title || (decodedSeed !== 'default' ? decodedSeed : 'Aurahub Video'),
      category: category || 'Video',
    });

    return new NextResponse(svg, {
      status: 200,
      headers: {
        'Content-Type': 'image/svg+xml; charset=utf-8',
        'Cache-Control': 'public, max-age=31536000, immutable',
      },
    });
  } catch (error) {
    console.error('Error in thumbnail generation route:', error);
    const fallbackSvg = generateThumbnailSvg({
      seed: 'aurahub-fallback',
      title: 'Aurahub',
      category: 'Video',
    });
    return new NextResponse(fallbackSvg, {
      status: 200,
      headers: {
        'Content-Type': 'image/svg+xml; charset=utf-8',
        'Cache-Control': 'public, max-age=3600',
      },
    });
  }
}
