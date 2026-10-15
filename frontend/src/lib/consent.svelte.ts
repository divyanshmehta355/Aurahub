// Content Mode & Consent Manager (Svelte 5 runes)

export const contentModeState = $state({
	isAdult: false,
	hasConsent: false,
	isLoaded: false
});

export function initContentMode() {
	if (typeof window !== 'undefined') {
		const storedConsent = localStorage.getItem('aurahub_adult_consent') === 'true';
		const storedMode = localStorage.getItem('aurahub_content_mode');
		contentModeState.hasConsent = storedConsent;
		// Adult content is only enabled if user explicitly chose adult mode AND consented
		contentModeState.isAdult = storedConsent && storedMode === 'adult';
		contentModeState.isLoaded = true;
	}
}

export function grantAdultConsent() {
	if (typeof window !== 'undefined') {
		localStorage.setItem('aurahub_adult_consent', 'true');
	}
	contentModeState.hasConsent = true;
}

export function revokeAdultConsent() {
	if (typeof window !== 'undefined') {
		localStorage.removeItem('aurahub_adult_consent');
		localStorage.setItem('aurahub_content_mode', 'general');
	}
	contentModeState.hasConsent = false;
	contentModeState.isAdult = false;
}

export function setContentMode(adult: boolean) {
	if (adult && !contentModeState.hasConsent) {
		grantAdultConsent();
	}
	if (typeof window !== 'undefined') {
		localStorage.setItem('aurahub_content_mode', adult ? 'adult' : 'general');
	}
	contentModeState.isAdult = adult;
}

export function toggleContentMode(onRequestConsent?: () => void) {
	if (!contentModeState.isAdult) {
		// Attempting to turn ON 18+ mode
		if (!contentModeState.hasConsent) {
			if (onRequestConsent) {
				onRequestConsent();
			}
			return;
		}
		setContentMode(true);
	} else {
		// Turning OFF 18+ mode (returning to safe general content)
		setContentMode(false);
	}
}

// Backwards-compatible aliases
export const consentState = contentModeState;
export const initAdultConsent = initContentMode;
