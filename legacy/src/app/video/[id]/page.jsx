import React from 'react';
import Video from '@/models/Video';
import dbConnect from '@/lib/dbConnect';
import VideoPlayerPageClient from './VideoPlayerPageClient';

export async function generateMetadata({ params }) {
  await dbConnect();
  try {
    const { id } = await params;
    const video = await Video.findById(id);
    if (!video) {
      return {
        title: 'Video Not Found',
        description: 'This video could not be found.',
      };
    }
    const pageDescription = video.description || `Watch ${video.title} on Aurahub`;
    return {
      title: `${video.title} - Aurahub`,
      description: pageDescription,
      keywords: video.tags ? video.tags.join(', ') : '',
      openGraph: {
        title: video.title,
        description: pageDescription,
        images: [
          {
            url: video.thumbnailUrl,
            width: 1200,
            height: 630,
          },
        ],
      },
    };
  } catch (error) {
    return {
      title: 'Server Error',
      description: 'An error occurred while fetching video data.',
    };
  }
}

const VideoPlayerPage = () => <VideoPlayerPageClient />;

export default VideoPlayerPage;