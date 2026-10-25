<script lang="ts">
	import { page } from '$app/state';
	import { fetchApi } from '#lib/api';
	import VideoThumbnail from '#lib/components/VideoThumbnail.svelte';
	import WatchLaterButton from '#lib/components/WatchLaterButton.svelte';
	import { PlaySquare } from 'lucide-svelte';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { Skeleton } from '#lib/components/ui/skeleton';
	import { Button } from '#lib/components/ui/button';

	const queryClient = useQueryClient();

	let currentPage = $derived(Number(page.url.searchParams.get('page')) || 1);
	const limit = 12;

	const subscriptionsQuery = createQuery(
		() => ({
			queryKey: ['subscriptions', currentPage],
			queryFn: async () => {
				const res = await fetchApi(`/videos/feed/subscriptions?page=${currentPage}&limit=${limit}`);
				return res;
			}
		}),
		() => queryClient
	);

	function getVisiblePages(current: number, total: number) {
		const pages = [];
		const maxVisible = 5;
		let start = Math.max(1, current - Math.floor(maxVisible / 2));
		let end = Math.min(total, start + maxVisible - 1);

		if (end - start + 1 < maxVisible) {
			start = Math.max(1, end - maxVisible + 1);
		}

		for (let i = start; i <= end; i++) {
			pages.push(i);
		}
		return pages;
	}

	function formatTimeAgo(dateString: string) {
		const date = new Date(dateString);
		const seconds = Math.floor((new Date().getTime() - date.getTime()) / 1000);
		let interval = seconds / 31536000;
		if (interval > 1) return Math.floor(interval) + ' years ago';
		interval = seconds / 2592000;
		if (interval > 1) return Math.floor(interval) + ' months ago';
		interval = seconds / 86400;
		if (interval > 1) return Math.floor(interval) + ' days ago';
		interval = seconds / 3600;
		if (interval > 1) return Math.floor(interval) + ' hours ago';
		interval = seconds / 60;
		if (interval > 1) return Math.floor(interval) + ' minutes ago';
		return Math.floor(seconds) + ' seconds ago';
	}
</script>

<svelte:head>
	<title>Subscriptions - Aurahub</title>
</svelte:head>

<div class="space-y-6 px-4 md:px-0">
	<div class="flex items-center gap-3 border-b border-slate-border/60 pt-2 pb-4 md:pt-0">
		<div class="flex h-10 w-10 items-center justify-center rounded-xl bg-aura-lavender/10 text-aura-lavender">
			<PlaySquare class="h-5 w-5" />
		</div>
		<div>
			<h1 class="text-xl font-bold tracking-tight text-soft-ivory md:text-2xl font-serif">Subscriptions</h1>
			<p class="text-xs text-muted-silver md:text-sm">Latest uploads from channels you follow</p>
		</div>
	</div>

	{#if subscriptionsQuery.isPending}
		<div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
			{#each Array(8) as _}
				<div class="flex animate-pulse flex-col space-y-3">
					<Skeleton class="aspect-video w-full rounded-2xl bg-deep-slate/60" />
					<div class="flex gap-3 px-1">
						<Skeleton class="h-10 w-10 shrink-0 rounded-full bg-deep-slate/60" />
						<div class="flex-1 space-y-2">
							<Skeleton class="h-4 w-[90%] rounded bg-deep-slate/60" />
							<Skeleton class="h-3 w-[60%] rounded bg-deep-slate/60" />
						</div>
					</div>
				</div>
			{/each}
		</div>
	{:else if subscriptionsQuery.isError}
		<div class="flex flex-col items-center justify-center py-12 text-center">
			<div class="rounded-2xl border border-destructive/30 bg-destructive/10 p-6 text-destructive">
				<p>{subscriptionsQuery.error.message || 'Failed to load subscriptions.'}</p>
				<Button variant="outline" class="mt-4 rounded-full border-destructive/40" onclick={() => subscriptionsQuery.refetch()}
					>Try again</Button
				>
			</div>
		</div>
	{:else if !subscriptionsQuery.data?.videos || subscriptionsQuery.data.videos.length === 0}
		<div
			class="mx-4 flex flex-col items-center justify-center rounded-3xl border border-dashed border-slate-border/80 bg-deep-slate/20 py-16 text-center md:mx-0 md:py-24"
		>
			<div class="mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-deep-slate text-muted-silver">
				<PlaySquare class="h-7 w-7 opacity-60" />
			</div>
			<h2 class="mb-2 text-lg font-bold text-soft-ivory md:text-xl">No new videos</h2>
			<p class="max-w-sm px-4 text-sm text-muted-silver md:text-base">
				The channels you subscribe to haven't uploaded any videos recently.
			</p>
			<Button variant="default" class="mt-6 rounded-full bg-aura-lavender text-midnight-ink font-semibold hover:bg-aura-lavender/90 shadow-md shadow-aura-lavender/20" href="/">Discover Channels</Button>
		</div>
	{:else}
		<div class="grid grid-cols-1 gap-5 sm:grid-cols-2 sm:gap-6 lg:grid-cols-3 xl:grid-cols-4">
			{#each (subscriptionsQuery.data.videos || []).filter((v: any) => !v.isShort) as video}
				<a
					href={`/watch/${video.fileId || video.id}`}
					class="group flex flex-col space-y-2.5 sm:space-y-3 min-w-0"
				>
					<!-- Thumbnail -->
					<div class="relative overflow-hidden rounded-2xl border border-slate-border/40 bg-deep-slate/40 shadow-sm transition-all duration-300 group-hover:border-aura-lavender/40 group-hover:shadow-lg">
						<VideoThumbnail
							{video}
							class="aspect-video w-full rounded-2xl"
							imgClass="transition-transform duration-300 group-hover:scale-105"
						>
							<WatchLaterButton videoId={video.id} />
						</VideoThumbnail>
					</div>
					<!-- Metadata -->
					<div class="flex gap-3 px-1 min-w-0">
						<div class="h-9 w-9 shrink-0 overflow-hidden rounded-full ring-2 ring-slate-border/60 transition-all group-hover:ring-aura-lavender/50 sm:h-10 sm:w-10">
							<img
								src={video.uploader?.avatar ||
									`https://api.dicebear.com/7.x/identicon/svg?seed=${video.uploader?.username}`}
								alt={video.uploader?.username}
								class="h-full w-full object-cover transition-transform group-hover:scale-110"
							/>
						</div>
						<div class="flex flex-col min-w-0 flex-1 overflow-hidden">
							<h3
								class="line-clamp-2 text-sm leading-tight font-semibold text-soft-ivory transition-colors group-hover:text-aura-lavender sm:text-base break-words [overflow-wrap:anywhere]"
								title={video.title}
							>
								{video.title}
							</h3>
							<p
								class="mt-1 text-xs text-muted-silver transition-colors hover:text-soft-ivory sm:text-sm"
							>
								{video.uploader?.username}
							</p>
							<div class="mt-0.5 text-[11px] text-muted-silver/80 sm:text-xs">
								<span>{video.views || 0} views</span>
								<span class="mx-1">•</span>
								<span
									>{formatTimeAgo(
										video.createdAt || video.created_at || new Date().toISOString()
									)}</span
								>
							</div>
						</div>
					</div>
				</a>
			{/each}
		</div>

		{#if subscriptionsQuery.data?.totalPages > 1}
			<div class="mt-10 flex justify-center pb-8 md:mt-14">
				<nav class="flex items-center gap-1.5 md:gap-2">
					<Button
						variant="outline"
						size="sm"
						disabled={currentPage === 1}
						href={`/subscriptions?page=${currentPage - 1}`}
						class="rounded-full border-slate-border text-xs font-semibold text-soft-ivory hover:bg-slate-elevated disabled:opacity-40"
					>
						Previous
					</Button>

					<div class="mx-2 hidden items-center gap-1.5 sm:flex">
						{#each getVisiblePages(currentPage, subscriptionsQuery.data.totalPages) as p}
							<Button
								variant={currentPage === p ? 'default' : 'ghost'}
								size="sm"
								href={`/subscriptions?page=${p}`}
								class={`h-8 w-8 rounded-full p-0 text-xs font-semibold transition-all ${
									currentPage === p
										? 'bg-aura-lavender text-midnight-ink hover:bg-aura-lavender/90 shadow-sm'
										: 'text-muted-silver hover:bg-slate-elevated hover:text-soft-ivory'
								}`}
							>
								{p}
							</Button>
						{/each}
					</div>

					<Button
						variant="outline"
						size="sm"
						disabled={currentPage === subscriptionsQuery.data.totalPages}
						href={`/subscriptions?page=${currentPage + 1}`}
						class="rounded-full border-slate-border text-xs font-semibold text-soft-ivory hover:bg-slate-elevated disabled:opacity-40"
					>
						Next
					</Button>
				</nav>
			</div>
		{/if}
	{/if}
</div>
