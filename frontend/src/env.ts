import { defineEnvVars } from '@sveltejs/kit/env';

export const variables = defineEnvVars({
	PUBLIC_API_URL: {
		public: true,
		schema: (val: string | undefined) => val
	}
});
