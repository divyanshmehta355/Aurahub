import { PUBLIC_API_URL } from '$app/env/public';
import { contentModeState } from '#lib/consent.svelte';

export const API_URL = PUBLIC_API_URL || '/api';

export async function fetchApi<T = any>(endpoint: string, options: RequestInit = {}): Promise<T> {
	const isFormData = options.body instanceof FormData;
	const headers: HeadersInit = { ...options.headers };

	if (!isFormData) {
		(headers as Record<string, string>)['Content-Type'] = 'application/json';
	}

	let finalEndpoint = endpoint;
	if (contentModeState.isAdult && !finalEndpoint.includes('adult=')) {
		const separator = finalEndpoint.includes('?') ? '&' : '?';
		finalEndpoint += `${separator}adult=true`;
	}

	const res = await fetch(`${API_URL}${finalEndpoint}`, {
		...options,
		headers,
		credentials: 'include'
	});

	if (!res.ok) {
		const errorData = await res.json().catch(() => ({ message: 'API Request failed' }));
		throw new Error(errorData.message || errorData.error || `API Error: ${res.status}`);
	}

	return res.json() as Promise<T>;
}
