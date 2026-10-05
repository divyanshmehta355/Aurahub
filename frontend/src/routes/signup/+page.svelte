<script lang="ts">
	import { Mail, Lock, User, ArrowRight, Loader2, Play } from 'lucide-svelte';
	import { fetchApi } from '#lib/api';
	import { createMutation } from '@tanstack/svelte-query';
	import { z } from 'zod';
	import { Button } from '#lib/components/ui/button';
	import { Input } from '#lib/components/ui/input';
	import { Label } from '#lib/components/ui/label';
	import * as Card from '#lib/components/ui/card';

	const signupSchema = z.object({
		username: z
			.string()
			.min(3, 'Username must be at least 3 characters')
			.max(20, 'Username too long'),
		email: z.string().email('Please enter a valid email address.'),
		password: z.string().min(6, 'Password must be at least 6 characters.')
	});

	let username = $state('');
	let email = $state('');
	let password = $state('');
	let validationErrors = $state<{ username?: string; email?: string; password?: string }>({});

	const signupMutation = createMutation(() => ({
		mutationFn: async (data: z.infer<typeof signupSchema>) => {
			return await fetchApi('/auth/register', {
				method: 'POST',
				body: JSON.stringify(data)
			});
		},
		onSuccess: () => {
			window.location.href = '/login';
		}
	}));

	function handleSignup(e: Event) {
		e.preventDefault();
		validationErrors = {};

		const result = signupSchema.safeParse({ username, email, password });
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

		signupMutation.mutate(result.data);
	}
</script>

<div class="mx-auto mt-8 w-full max-w-md px-4 md:mt-16">
	<Card.Root class="border-border shadow-sm">
		<Card.Header class="items-center space-y-1 pb-8">
			<div class="text-primary-foreground mb-4 rounded-xl bg-primary p-3">
				<Play class="h-7 w-7 fill-current" />
			</div>
			<Card.Title class="text-center text-2xl font-bold tracking-tight"
				>Create an account</Card.Title
			>
			<Card.Description class="text-center"
				>Join Aurahub to start sharing and discovering videos</Card.Description
			>
		</Card.Header>

		<Card.Content>
			{#if signupMutation.isError}
				<div
					class="bg-destructive/10 text-destructive mb-6 rounded-md p-3 text-center text-sm font-medium"
				>
					{signupMutation.error.message || 'Signup failed. Please try again.'}
				</div>
			{/if}

			<form onsubmit={handleSignup} class="space-y-4">
				<div class="space-y-2">
					<Label for="username">Username</Label>
					<div class="relative">
						<User class="text-muted-foreground absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2" />
						<Input
							id="username"
							type="text"
							bind:value={username}
							placeholder="johndoe"
							class="pl-10 {validationErrors.username
								? 'border-destructive focus-visible:ring-destructive'
								: ''}"
						/>
					</div>
					{#if validationErrors.username}
						<p class="text-destructive text-xs font-medium">{validationErrors.username}</p>
					{/if}
				</div>

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
					<Label for="password">Password</Label>
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

				<Button type="submit" disabled={signupMutation.isPending} class="mt-6 w-full">
					{#if signupMutation.isPending}
						<Loader2 class="mr-2 -ml-1 h-4 w-4 animate-spin" />
						Creating account...
					{:else}
						Sign Up
						<ArrowRight class="ml-2 h-4 w-4" />
					{/if}
				</Button>
			</form>
		</Card.Content>

		<Card.Footer class="flex justify-center border-t p-6">
			<div class="text-sm">
				<span class="text-muted-foreground">Already have an account? </span>
				<a href="/login" class="font-medium text-primary hover:underline">Sign in instead</a>
			</div>
		</Card.Footer>
	</Card.Root>
</div>
