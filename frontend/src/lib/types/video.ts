import type { UserSummary } from './user';

export type VideoVisibility = 'public' | 'unlisted' | 'private';

export interface Video {
	id: string;
	title: string;
	description?: string;
	thumbnailUrl?: string;
	streamtapeUrl?: string;
	fileId?: string;
	views?: number;
	likes?: number;
	dislikes?: number;
	isShort?: boolean;
	isAdult?: boolean;
	category?: string;
	tags?: string[];
	visibility?: VideoVisibility;
	createdAt?: string;
	uploader?: UserSummary;
	uploaderId?: string;
}

export interface VideoDetail extends Video {
	liked?: boolean;
	saved?: boolean;
	isUploader?: boolean;
}

export interface VideoListResponse {
	videos: Video[];
	total?: number;
	page?: number;
	limit?: number;
	hasMore?: boolean;
}
