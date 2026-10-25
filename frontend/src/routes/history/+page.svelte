<script lang="ts">
	import { fetchApi } from '#lib/api';
	import VideoThumbnail from '#lib/components/VideoThumbnail.svelte';
	import WatchLaterButton from '#lib/components/WatchLaterButton.svelte';
	import { Clock, Trash2 } from 'lucide-svelte';
	import { createQuery, createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { Skeleton } from '#lib/components/ui/skeleton';
	import { Button } from '#lib/components/ui/button';

	const queryClient = useQueryClient();

	const historyQuery = createQuery(
		() => ({
			queryKey: ['history'],
			queryFn: async () => {
				const res = await fetchApi('/user/history');
				return res.history || res.videos || res.items || res || [];
			}
		}),
		() => queryClient
	);

	const deleteHistoryMutation = createMutation(
		() => ({
			mutationFn: async (videoId: string) => {
				await fetchApi(`/user/history/${videoId}`, { method: 'DELETE' });
			},
			onSuccess: () => {
				queryClient.invalidateQueries({ queryKey: ['history'] });
			}
		}),
		() => queryClient
	);

	const clearHistoryMutation = createMutation(
		() => ({
			mutationFn: async () => {
				await fetchApi('/user/history', { method: 'DELETE' });
			},
			onSuccess: () => {
				queryClient.invalidateQueries({ queryKey: ['history'] });
			}
		}),
		() => queryClient
	);

	function handleDelete(e: Event, videoId: string) {
		e.preventDefault();
		e.stopPropagation();
		deleteHistoryMutation.mutate(videoId);
	}

	function handleClearAll() {
		if (confirm('Are you sure you want to clear your entire watch history?')) {
			clearHistoryMutation.mutate();
		}
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
	<title>Watch History - Aurahub</title>
</svelte:head>

<div class="space-y-4 px-4 md:space-y-6 md:px-0">
	<div class="flex items-center justify-between border-b border-slate-border/60 pt-2 pb-4 md:pt-0">
		<div class="flex items-center gap-3">
			<div class="flex h-10 w-10 items-center justify-center rounded-xl bg-aura-lavender/10 text-aura-lavender">
				<Clock class="h-5 w-5" />
			</div>
			<div>
				<h1 class="text-xl font-bold tracking-tight text-soft-ivory md:text-2xl font-serif">Watch History</h1>
				<p class="text-xs text-muted-silver md:text-sm">Manage videos you have previously watched</p>
			</div>
		</div>

		{#if historyQuery.data && historyQuery.data.length > 0}
			<Button
				variant="outline"
				size="sm"
				class="rounded-full border-slate-border/80 text-muted-silver hover:border-destructive/40 hover:bg-destructive/10 hover:text-destructive transition-all"
				onclick={handleClearAll}
				disabled={clearHistoryMutation.isPending}
			>
				Clear All
			</Button>
		{/if}
	</div>

	{#if historyQuery.isPending}
		<div class="flex max-w-4xl flex-col gap-4">
			{#each Array(5) as _}
				<div class="flex animate-pulse flex-col gap-4 rounded-2xl border border-slate-border/30 bg-deep-slate/30 p-3 sm:flex-row">
					<Skeleton class="aspect-video w-full shrink-0 rounded-xl bg-slate-elevated sm:w-64" />
					<div class="flex-1 space-y-3 py-2">
						<Skeleton class="h-5 w-3/4 rounded-lg bg-slate-elevated" />
						<Skeleton class="h-4 w-1/4 rounded-lg bg-slate-elevated" />
					</div>
				</div>
			{/each}
		</div>
	{:else if historyQuery.isError}
		<div class="flex flex-col items-center justify-center py-12 text-center">
			<div class="rounded-2xl border border-destructive/30 bg-destructive/10 p-6 text-destructive">
				<p>{historyQuery.error.message || 'Failed to load history.'}</p>
				<Button variant="outline" class="mt-4 rounded-full border-destructive/40" onclick={() => historyQuery.refetch()}
					>Try again</Button
				>
			</div>
		</div>
	{:else if !historyQuery.data || historyQuery.data.length === 0}
		<div
			class="mx-4 flex flex-col items-center justify-center rounded-3xl border border-dashed border-slate-border/80 bg-deep-slate/20 py-16 text-center md:mx-0 md:py-24"
		>
			<div class="mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-deep-slate text-muted-silver">
				<Clock class="h-7 w-7 opacity-60" />
			</div>
			<h2 class="mb-2 text-lg font-bold text-soft-ivory md:text-xl">No watch history</h2>
			<p class="max-w-sm px-4 text-sm text-muted-silver md:text-base">
				Videos you watch will show up here.
			</p>
			<Button variant="default" class="mt-6 rounded-full bg-aura-lavender text-midnight-ink font-semibold hover:bg-aura-lavender/90 shadow-md shadow-aura-lavender/20" href="/">Watch Videos</Button>
		</div>
	{:else}
		<div class="flex max-w-4xl flex-col gap-3 md:gap-3">
			{#each historyQuery.data as item}
				{@const video = item.video || item}
				<a
					href={`/watch/${video.fileId || video.id}`}
					class="group relative flex flex-col gap-3 rounded-2xl border border-slate-border/30 bg-deep-slate/30 p-2.5 transition-all hover:border-aura-lavender/40 hover:bg-deep-slate/80 sm:flex-row sm:p-3 md:gap-4 min-w-0"
				>
					<!-- Thumbnail -->
					<div class="w-full shrink-0 sm:w-64">
						<VideoThumbnail
							{video}
							class="aspect-video rounded-xl overflow-hidden border border-slate-border/40"
							imgClass="transition-transform duration-300 group-hover:scale-105"
						>
							<WatchLaterButton videoId={video.id} />
						</VideoThumbnail>
					</div>
					<!-- Metadata -->
					<div class="relative flex flex-1 flex-col px-1 py-1 sm:px-0 min-w-0 overflow-hidden">
						<div class="pr-8 min-w-0">
							<h3
								class="mb-1 line-clamp-2 text-base leading-tight font-semibold text-soft-ivory transition-colors group-hover:text-aura-lavender md:text-lg break-words [overflow-wrap:anywhere]"
								title={video.title}
							>
								{video.title}
							</h3>

							<div class="mb-1.5 flex items-center gap-2">
								<p class="text-[13px] font-medium text-muted-silver transition-colors group-hover:text-soft-ivory md:text-sm">
									{video.uploader_username || video.uploader?.username}
								</p>
							</div>

							<div class="mb-3 text-[11px] text-muted-silver/80 md:text-xs">
								<span>{video.views || 0} views</span>
								<span class="mx-1.5">•</span>
								<span
									>Watched {formatTimeAgo(
										item.watchedAt || item.watched_at || new Date().toISOString()
									)}</span
								>
							</div>

							{#if video.description}
								<p
									class="line-clamp-1 text-[11px] text-muted-silver/70 md:line-clamp-2 md:text-xs"
								>
									{video.description}
								</p>
							{/if}
						</div>

						<Button
							variant="ghost"
							size="icon"
							class="text-muted-silver hover:text-destructive hover:bg-destructive/10 absolute top-0 right-0 h-8 w-8 rounded-full opacity-100 transition-all group-hover:opacity-100 md:opacity-0"
							onclick={(e: MouseEvent) => handleDelete(e, video.id || video.fileId)}
							title="Remove from Watch History"
						>
							<Trash2 class="h-4 w-4" />
						</Button>
					</div>
				</a>
			{/each}
		</div>
	{/if}
</div>
