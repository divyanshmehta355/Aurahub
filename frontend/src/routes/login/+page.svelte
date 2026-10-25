<script lang="ts">
	import { Mail, Lock, ArrowRight, Loader2, Play } from 'lucide-svelte';
	import { fetchApi } from '#lib/api';
	import { createMutation } from '@tanstack/svelte-query';
	import { z } from 'zod';
	import { Button } from '#lib/components/ui/button';
	import { Input } from '#lib/components/ui/input';
	import { Label } from '#lib/components/ui/label';
	import * as Card from '#lib/components/ui/card';

	const loginSchema = z.object({
		email: z.string().email('Please enter a valid email address.'),
		password: z.string().min(1, 'Password is required.')
	});

	let email = $state('');
	let password = $state('');
	let validationErrors = $state<{ email?: string; password?: string }>({});

	const loginMutation = createMutation(() => ({
		mutationFn: async (data: z.infer<typeof loginSchema>) => {
			return await fetchApi('/auth/login', {
				method: 'POST',
				body: JSON.stringify(data)
			});
		},
		onSuccess: () => {
			window.location.href = '/';
		}
	}));

	function handleLogin(e: Event) {
		e.preventDefault();
		validationErrors = {};

		const result = loginSchema.safeParse({ email, password });
		if (!result.success) {
			const errors: Record<string, string> = {};
			const issues = (result.error as any).issues || (result.error as any).errors || [];
			for (const err of issues) {
				if (err.path && err.path[0]) {
					errors[err.path[0].toString()] = err.message;
				}
			}
			validationErrors = errors;
			return;
		}

		loginMutation.mutate(result.data);
	}
</script>

<div class="mx-auto mt-10 w-full max-w-md px-4 md:mt-20">
	<Card.Root class="rounded-3xl border border-border/70 bg-card shadow-2xl shadow-black/40 overflow-hidden">
		<Card.Header class="items-center space-y-1.5 pb-6 pt-8 text-center">
			<div class="mb-3 flex h-14 w-14 items-center justify-center rounded-2xl bg-primary text-primary-foreground shadow-lg shadow-primary/25">
				<svg viewBox="0 0 24 24" class="h-7 w-7 fill-current" aria-hidden="true">
					<circle cx="12" cy="12" r="3" />
					<path
						d="M12 2v3m0 14v3M2 12h3m14 0h3m-3.5-6.5-2.1 2.1m-8.8 8.8-2.1 2.1m0-13 2.1 2.1m8.8 8.8 2.1 2.1"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						fill="none"
					/>
				</svg>
			</div>
			<Card.Title class="text-center font-serif text-3xl font-bold tracking-tight text-foreground">Welcome Back</Card.Title>
			<Card.Description class="text-center text-xs sm:text-sm text-muted-foreground"
				>Sign in to your Aurahub cinema pass</Card.Description
			>
		</Card.Header>

		<Card.Content class="px-6 pb-6">
			{#if loginMutation.isError}
				<div
					class="bg-destructive/10 border border-destructive/20 text-destructive mb-6 rounded-xl p-3 text-center text-xs font-semibold"
				>
					{loginMutation.error.message || 'Login failed. Please verify credentials.'}
				</div>
			{/if}

			<form onsubmit={handleLogin} class="space-y-4">
				<div class="space-y-1.5">
					<Label for="email" class="text-xs font-semibold text-foreground">Email Address</Label>
					<div class="relative">
						<Mail class="text-muted-foreground absolute top-1/2 left-3.5 h-4 w-4 -translate-y-1/2" />
						<Input
							id="email"
							type="email"
							bind:value={email}
							placeholder="name@example.com"
							class="pl-10 h-10 rounded-xl bg-secondary/50 border-border/70 focus-visible:border-primary/60 focus-visible:ring-2 focus-visible:ring-primary/20 text-sm text-foreground {validationErrors.email
								? 'border-destructive focus-visible:ring-destructive'
								: ''}"
						/>
					</div>
					{#if validationErrors.email}
						<p class="text-destructive text-xs font-medium">{validationErrors.email}</p>
					{/if}
				</div>

				<div class="space-y-1.5">
					<div class="flex items-center justify-between">
						<Label for="password" class="text-xs font-semibold text-foreground">Password</Label>
						<a href="/forgot-password" class="text-xs font-medium text-primary hover:underline"
							>Forgot password?</a
						>
					</div>
					<div class="relative">
						<Lock class="text-muted-foreground absolute top-1/2 left-3.5 h-4 w-4 -translate-y-1/2" />
						<Input
							id="password"
							type="password"
							bind:value={password}
							placeholder="••••••••"
							class="pl-10 h-10 rounded-xl bg-secondary/50 border-border/70 focus-visible:border-primary/60 focus-visible:ring-2 focus-visible:ring-primary/20 text-sm text-foreground {validationErrors.password
								? 'border-destructive focus-visible:ring-destructive'
								: ''}"
						/>
					</div>
					{#if validationErrors.password}
						<p class="text-destructive text-xs font-medium">{validationErrors.password}</p>
					{/if}
				</div>

				<Button type="submit" disabled={loginMutation.isPending} class="mt-6 w-full h-10 rounded-full font-semibold shadow-md shadow-primary/25">
					{#if loginMutation.isPending}
						<Loader2 class="mr-2 -ml-1 h-4 w-4 animate-spin" />
						Signing in...
					{:else}
						Sign In
						<ArrowRight class="ml-2 h-4 w-4" />
					{/if}
				</Button>
			</form>
		</Card.Content>

		<Card.Footer class="flex justify-center border-t border-border/50 py-5 bg-secondary/20">
			<div class="text-xs sm:text-sm">
				<span class="text-muted-foreground">Don't have an account? </span>
				<a href="/signup" class="font-semibold text-primary hover:underline">Create an account</a>
			</div>
		</Card.Footer>
	</Card.Root>
</div>
