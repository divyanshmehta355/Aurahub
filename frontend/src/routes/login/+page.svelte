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
			for (const err of result.error.errors) {
				if (err.path[0]) {
					errors[err.path[0].toString()] = err.message;
				}
			}
			validationErrors = errors;
			return;
		}

		loginMutation.mutate(result.data);
	}
</script>

<div class="mx-auto mt-12 w-full max-w-md px-4 md:mt-24">
	<Card.Root class="border-border shadow-sm">
		<Card.Header class="items-center space-y-1 pb-8">
			<div class="text-primary-foreground mb-4 rounded-xl bg-primary p-3">
				<Play class="h-7 w-7 fill-current" />
			</div>
			<Card.Title class="text-center text-2xl font-bold tracking-tight">Welcome back</Card.Title>
			<Card.Description class="text-center"
				>Enter your credentials to access your account</Card.Description
			>
		</Card.Header>

		<Card.Content>
			{#if loginMutation.isError}
				<div
					class="bg-destructive/10 text-destructive mb-6 rounded-md p-3 text-center text-sm font-medium"
				>
					{loginMutation.error.message || 'Login failed. Please try again.'}
				</div>
			{/if}

			<form onsubmit={handleLogin} class="space-y-4">
				<div class="space-y-2">
					<Label for="email">Email address</Label>
					<div class="relative">
						<Mail class="text-muted-foreground absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2" />
						<Input
							id="email"
							type="email"
							bind:value={email}
							placeholder="name@example.com"
							class="pl-10 {validationErrors.email
								? 'border-destructive focus-visible:ring-destructive'
								: ''}"
						/>
					</div>
					{#if validationErrors.email}
						<p class="text-destructive text-xs font-medium">{validationErrors.email}</p>
					{/if}
				</div>

				<div class="space-y-2">
					<div class="flex items-center justify-between">
						<Label for="password">Password</Label>
						<a href="/forgot-password" class="text-xs font-medium text-primary hover:underline"
							>Forgot password?</a
						>
					</div>
					<div class="relative">
						<Lock class="text-muted-foreground absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2" />
						<Input
							id="password"
							type="password"
							bind:value={password}
							placeholder="••••••••"
							class="pl-10 {validationErrors.password
								? 'border-destructive focus-visible:ring-destructive'
								: ''}"
						/>
					</div>
					{#if validationErrors.password}
						<p class="text-destructive text-xs font-medium">{validationErrors.password}</p>
					{/if}
				</div>

				<Button type="submit" disabled={loginMutation.isPending} class="mt-6 w-full">
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

		<Card.Footer class="flex justify-center border-t p-6">
			<div class="text-sm">
				<span class="text-muted-foreground">Don't have an account? </span>
				<a href="/signup" class="font-medium text-primary hover:underline">Sign up now</a>
			</div>
		</Card.Footer>
	</Card.Root>
</div>
