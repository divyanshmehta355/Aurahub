export interface User {
	id: string;
	username: string;
	email: string;
	avatar?: string;
	banner?: string;
	bio?: string;
	showAdultContent?: boolean;
	isAdmin?: boolean;
}

export interface UserSummary {
	id: string;
	username: string;
	avatar?: string;
	subscriberCount?: number;
}

export interface UserProfile extends User {
	subscriberCount: number;
	videoCount?: number;
	isSubscribed?: boolean;
}
