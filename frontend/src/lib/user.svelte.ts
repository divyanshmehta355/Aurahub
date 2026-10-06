export const userState = $state({
	user: null as any | null,
	isLoaded: false
});

export function setUser(user: any) {
	userState.user = user;
	userState.isLoaded = true;
	if (typeof window !== 'undefined' && user && typeof user.showAdultContent === 'boolean') {
		localStorage.setItem('showAdultContent', user.showAdultContent ? 'true' : 'false');
	}
}

export function clearUser() {
	userState.user = null;
	userState.isLoaded = true;
}
