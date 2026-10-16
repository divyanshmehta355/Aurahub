import type { User } from './types';

export const userState = $state<{
	user: User | null;
	isLoaded: boolean;
}>({
	user: null,
	isLoaded: false
});

export function setUser(user: User) {
	userState.user = user;
	userState.isLoaded = true;
	if (typeof window !== 'undefined') {
		localStorage.removeItem('showAdultContent');
	}
}

export function clearUser() {
	userState.user = null;
	userState.isLoaded = true;
}
