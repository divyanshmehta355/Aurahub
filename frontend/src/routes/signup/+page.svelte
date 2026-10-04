<script lang="ts">
	import { Mail, Lock, User, ArrowRight } from 'lucide-svelte';
	import { fetchApi } from '#lib/api';
	
	let username = $state('');
	let email = $state('');
	let password = $state('');
	let isLoading = $state(false);
	let errorMsg = $state('');

	async function handleSignup(e: Event) {
		e.preventDefault();
		isLoading = true;
		errorMsg = '';
		
		try {
			const res = await fetchApi('/auth/register', {
				method: 'POST',
				body: JSON.stringify({ username, email, password })
			});
			
			// Registration successful, usually they need to login or are automatically logged in
			// In our backend, register doesn't seem to set a cookie automatically, so let's redirect to login
			window.location.href = '/login';
		} catch (err: any) {
			errorMsg = err.message || 'Signup failed. Please try again.';
		} finally {
			isLoading = false;
		}
	}
</script>

<div class="max-w-md mx-auto w-full mt-8 md:mt-16 p-6 sm:p-8 bg-card border rounded-2xl shadow-sm">
	<div class="flex flex-col items-center mb-8">
		<div class="bg-primary text-primary-foreground p-3 rounded-xl mb-4">
			<svg xmlns="http://www.w3.org/2000/svg" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-play"><polygon points="5 3 19 12 5 21 5 3"/></svg>
		</div>
		<h1 class="text-2xl font-bold text-center tracking-tight">Create an account</h1>
		<p class="text-sm text-muted-foreground mt-2 text-center">Join Aurahub to start sharing and discovering videos</p>
	</div>

	{#if errorMsg}
		<div class="p-3 mb-4 rounded-md bg-red-100 dark:bg-red-900/30 text-red-600 dark:text-red-400 text-sm text-center">
			{errorMsg}
		</div>
	{/if}

	<form onsubmit={handleSignup} class="space-y-4">
		<div class="space-y-2">
			<label for="username" class="text-sm font-medium leading-none">Username</label>
			<div class="relative">
				<User class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
				<input
					id="username"
					type="text"
					bind:value={username}
					required
					placeholder="johndoe"
					class="flex h-10 w-full rounded-md border border-input bg-transparent px-3 py-2 pl-10 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
				/>
			</div>
		</div>

		<div class="space-y-2">
			<label for="email" class="text-sm font-medium leading-none">Email address</label>
			<div class="relative">
				<Mail class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
				<input
					id="email"
					type="email"
					bind:value={email}
					required
					placeholder="name@example.com"
					class="flex h-10 w-full rounded-md border border-input bg-transparent px-3 py-2 pl-10 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
				/>
			</div>
		</div>

		<div class="space-y-2">
			<label for="password" class="text-sm font-medium leading-none">Password</label>
			<div class="relative">
				<Lock class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
				<input
					id="password"
					type="password"
					bind:value={password}
					required
					placeholder="••••••••"
					class="flex h-10 w-full rounded-md border border-input bg-transparent px-3 py-2 pl-10 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
				/>
			</div>
		</div>

		<button
			type="submit"
			disabled={isLoading}
			class="inline-flex w-full items-center justify-center rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 bg-primary text-primary-foreground hover:bg-primary/90 h-10 px-4 py-2 mt-6"
		>
			{#if isLoading}
				<svg class="animate-spin -ml-1 mr-2 h-4 w-4 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
				Creating account...
			{:else}
				Sign Up
				<ArrowRight class="ml-2 h-4 w-4" />
			{/if}
		</button>
	</form>

	<div class="mt-6 text-center text-sm">
		<span class="text-muted-foreground">Already have an account? </span>
		<a href="/login" class="font-medium text-primary hover:underline">Sign in</a>
	</div>
</div>
