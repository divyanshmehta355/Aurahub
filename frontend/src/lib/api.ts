import { PUBLIC_API_URL } from '$app/env/public';

export const API_URL = PUBLIC_API_URL || '/api';

export async function fetchApi(endpoint: string, options: RequestInit = {}) {
	const isFormData = options.body instanceof FormData;
	const headers: HeadersInit = { ...options.headers };

	if (!isFormData) {
		(headers as Record<string, string>)['Content-Type'] = 'application/json';
	}

	const res = await fetch(`${API_URL}${endpoint}`, {
		...options,
		headers,
		credentials: 'include'
	});

	if (!res.ok) {
		const errorData = await res.json().catch(() => ({ message: 'API Request failed' }));
		throw new Error(errorData.message || `API Error: ${res.status}`);
	}

	return res.json();
}
