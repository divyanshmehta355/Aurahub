<script lang="ts">
	import { fetchApi } from '#lib/api';
	import { userState } from '#lib/user.svelte';
	import { BarChart, Video } from 'lucide-svelte';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { Button } from '#lib/components/ui/button';
	import { Skeleton } from '#lib/components/ui/skeleton';
	import * as Card from '#lib/components/ui/card';

	const queryClient = useQueryClient();

	const analyticsQuery = createQuery(
		() => ({
			queryKey: ['analytics'],
			queryFn: async () => {
				if (!userState.user && userState.isLoaded) {
					window.location.href = '/login';
					return null;
				}
				return await fetchApi('/creator/analytics').catch(() => null);
			}
		}),
		() => queryClient
	);
</script>

<svelte:head>
	<title>Analytics - Aurahub</title>
</svelte:head>

<div class="mx-auto max-w-6xl space-y-6 px-4 py-4 md:space-y-8 md:px-6 md:py-6">
	<div
		class="flex flex-col justify-between gap-4 border-b pb-4 md:flex-row md:items-end md:border-0 md:pb-0"
	>
		<div>
			<h1 class="text-2xl font-bold tracking-tight md:text-3xl">Channel Analytics</h1>
			<p class="text-muted-foreground mt-1 text-sm md:text-base">
				View your channel performance and growth.
			</p>
		</div>
		<Button href="/dashboard" class="w-full md:w-auto">
			<Video class="mr-2 h-4 w-4" />
			Manage Content
		</Button>
	</div>

	<div class="min-h-[400px]">
		{#if analyticsQuery.isPending}
			<div class="mb-8 grid grid-cols-1 gap-4 md:grid-cols-3 md:gap-6">
				<Skeleton class="h-32 rounded-xl" />
				<Skeleton class="h-32 rounded-xl" />
				<Skeleton class="h-32 rounded-xl" />
			</div>
			<Skeleton class="h-48 w-full rounded-xl" />
		{:else if analyticsQuery.isError}
			<div class="flex flex-col items-center justify-center py-12 text-center">
				<div class="bg-destructive/10 text-destructive rounded-xl p-4">
					<p>{analyticsQuery.error?.message || 'Failed to load analytics data.'}</p>
					<Button variant="outline" class="mt-4" onclick={() => analyticsQuery.refetch()}
						>Try again</Button
					>
				</div>
			</div>
		{:else if analyticsQuery.data}
			<div class="mb-6 grid grid-cols-1 gap-4 sm:grid-cols-2 md:mb-8 md:grid-cols-3 md:gap-6">
				<Card.Root>
					<Card.Header class="pb-2">
						<Card.Title class="text-muted-foreground text-sm font-medium">Total Views</Card.Title>
					</Card.Header>
					<Card.Content>
						<div class="text-3xl font-bold">{analyticsQuery.data.total_views || 0}</div>
					</Card.Content>
				</Card.Root>

				<Card.Root>
					<Card.Header class="pb-2">
						<Card.Title class="text-muted-foreground text-sm font-medium">Total Likes</Card.Title>
					</Card.Header>
					<Card.Content>
						<div class="text-3xl font-bold">{analyticsQuery.data.total_likes || 0}</div>
					</Card.Content>
				</Card.Root>

				<Card.Root class="sm:col-span-2 md:col-span-1">
					<Card.Header class="pb-2">
						<Card.Title class="text-muted-foreground text-sm font-medium">Total Comments</Card.Title
						>
					</Card.Header>
					<Card.Content>
						<div class="text-3xl font-bold">{analyticsQuery.data.total_comments || 0}</div>
					</Card.Content>
				</Card.Root>
			</div>

			<Card.Root
				class="flex min-h-[150px] items-center justify-center border-dashed md:min-h-[200px]"
			>
				<Card.Content class="pt-6 text-center">
					<BarChart class="text-muted-foreground/50 mx-auto mb-3 h-8 w-8" />
					<p class="text-muted-foreground max-w-sm text-sm">
						More detailed analytics charts will appear here as your channel grows.
					</p>
				</Card.Content>
			</Card.Root>
		{:else}
			<Card.Root class="flex min-h-[200px] items-center justify-center border-dashed">
				<Card.Content class="pt-6 text-center">
					<p class="text-muted-foreground">
						No analytics data available yet. Upload videos to get started!
					</p>
					<Button variant="default" class="mt-4 rounded-full" href="/upload">Upload Video</Button>
				</Card.Content>
			</Card.Root>
		{/if}
	</div>
</div>
