import type { Video } from './video';

export interface Playlist {
	id: string;
	title: string;
	description?: string;
	isPublic?: boolean;
	ownerId?: string;
	videoCount?: number;
	videos?: Video[];
	createdAt?: string;
}

export interface PlaylistResponse {
	playlist: Playlist;
	videos: Video[];
}
