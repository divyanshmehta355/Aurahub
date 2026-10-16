import type { UserSummary } from './user';

export interface Comment {
	id: string;
	text: string;
	content?: string;
	authorId: string;
	author?: UserSummary;
	videoId: string;
	parentCommentId?: string | null;
	createdAt: string;
	likes?: number;
	replyCount?: number;
	replies?: Comment[];
}

export interface CreateCommentRequest {
	videoId: string;
	text: string;
	parentCommentId?: string;
}
