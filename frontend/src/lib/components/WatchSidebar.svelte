<script lang="ts">
	import { Play, Loader2, RotateCw } from 'lucide-svelte';
	import VideoThumbnail from '#lib/components/VideoThumbnail.svelte';
	import WatchLaterButton from '#lib/components/WatchLaterButton.svelte';
	import { Skeleton } from '#lib/components/ui/skeleton';
	import { Button } from '#lib/components/ui/button';
	import { fetchApi } from '#lib/api';
	import { contentModeState } from '#lib/consent.svelte';
	import { createQuery, createInfiniteQuery, useQueryClient } from '@tanstack/svelte-query';

	let {
		videoId = '',
		playlistId = null,
		isTheaterMode = false
	}: {
		videoId?: string;
		playlistId?: string | null;
		isTheaterMode?: boolean;
	} = $props();

	const queryClient = useQueryClient();

	function generateSeed(): string {
		return Math.random().toString(36).substring(2, 10);
	}

	let sidebarSeed = $state<string>(generateSeed());

	function refreshSuggestions() {
		sidebarSeed = generateSeed();
		queryClient.invalidateQueries({ queryKey: ['suggestions', videoId] });
	}

	const playlistQuery = createQuery(
		() => ({
			queryKey: ['playlist', playlistId],
			queryFn: async () => {
				const res = await fetchApi(`/playlists/${playlistId}`);
				const p = res.playlist || res;
				if (res.videos) p.videos = res.videos;
				return p;
			},
			enabled: !!playlistId
		}),
		() => queryClient
	);

	const suggestionsQuery = createInfiniteQuery(
		() => ({
			queryKey: ['suggestions', videoId, contentModeState.isAdult, sidebarSeed],
			queryFn: async ({ pageParam = 1 }) => {
				const res = await fetchApi(
					`/videos/suggestions?exclude=${videoId}&limit=10&page=${pageParam}&seed=${sidebarSeed}`
				);
				return res;
			},
			getNextPageParam: (lastPage: any) => {
				return lastPage.currentPage < lastPage.totalPages ? lastPage.currentPage + 1 : undefined;
			},
			initialPageParam: 1
		}),
		() => queryClient
	);

	function infiniteScroll(node: HTMLElement) {
		const observer = new IntersectionObserver(
			(entries) => {
				if (entries[0].isIntersecting) {
					if (suggestionsQuery.hasNextPage && !suggestionsQuery.isFetchingNextPage) {
						suggestionsQuery.fetchNextPage();
					}
				}
			},
			{ rootMargin: '200px' }
		);
		observer.observe(node);
		return {
			destroy() {
				observer.disconnect();
			}
		};
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

<div
	class={`col-span-1 flex h-full flex-col space-y-4 transition-all duration-300 min-w-0 ${
		isTheaterMode ? 'lg:col-span-1 lg:col-start-3 lg:row-start-2' : 'lg:col-span-1'
	}`}
>
	{#if playlistQuery.isPending && playlistId}
		<Skeleton class="h-64 rounded-2xl" />
	{:else if playlistQuery.data && playlistQuery.data.videos}
		<div class="bg-card flex max-h-[500px] flex-col rounded-2xl border border-border/70 shadow-sm min-w-0 overflow-hidden">
			<div class="bg-secondary/40 border-b border-border/50 p-4 min-w-0">
				<h3 class="line-clamp-1 text-base font-bold text-foreground break-words [overflow-wrap:anywhere]">{playlistQuery.data.title}</h3>
				<p class="text-muted-foreground mt-1 text-xs">
					{playlistQuery.data.owner?.username || 'Playlist'} • {playlistQuery.data.videos.findIndex(
						(v: any) => v.fileId === videoId || v.id === videoId
					) + 1} / {playlistQuery.data.videos.length}
				</p>
			</div>
			<div class="custom-scrollbar flex-1 overflow-y-auto">
				{#each playlistQuery.data.videos as pv, i}
					<a
						href={`/watch/${pv.fileId || pv.id}?list=${playlistId}`}
						class={`hover:bg-secondary/60 flex gap-3 p-3 transition-colors min-w-0 ${pv.fileId === videoId || pv.id === videoId ? 'border-primary bg-primary/10 border-l-2 text-primary font-semibold' : ''}`}
					>
						<div
							class="text-muted-foreground flex w-5 shrink-0 items-center justify-center text-xs"
						>
							{#if pv.fileId === videoId || pv.id === videoId}
								<Play class="fill-primary text-primary h-3.5 w-3.5" />
							{:else}
								<span class="text-muted-silver">{i + 1}</span>
							{/if}
						</div>
						<VideoThumbnail video={pv} class="h-[72px] w-32 shrink-0 rounded-xl border border-border/60" />
						<div class="flex flex-1 flex-col justify-start overflow-hidden min-w-0">
							<h4
								class={`line-clamp-2 text-xs leading-snug font-semibold break-words [overflow-wrap:anywhere] ${pv.fileId === videoId || pv.id === videoId ? 'text-primary' : 'text-foreground'}`}
							>
								{pv.title}
							</h4>
							<span class="text-muted-foreground mt-1 text-[11px]">
								{pv.uploader?.username || 'Unknown'}
							</span>
						</div>
					</a>
				{/each}
			</div>
		</div>
	{/if}

	<div class="mt-1 flex items-center justify-between border-b border-border/40 pb-2">
		<h3 class="text-base font-bold text-foreground">Up Next</h3>
		<Button
			variant="ghost"
			size="sm"
			class="h-7 w-7 p-0 rounded-full text-muted-foreground hover:text-foreground hover:bg-secondary/60"
			onclick={refreshSuggestions}
			title="Shuffle Up Next recommendations"
		>
			<RotateCw class="h-3.5 w-3.5 {suggestionsQuery.isFetching ? 'animate-spin' : ''}" />
		</Button>
	</div>

	{#if suggestionsQuery.isPending}
		<div class="flex flex-col gap-3">
			{#each Array(6) as _}
				<div class="flex gap-3">
					<Skeleton class="h-22 w-36 rounded-xl shrink-0" />
					<div class="flex-1 space-y-2 py-1">
						<Skeleton class="h-4 w-full rounded-lg" />
						<Skeleton class="h-3 w-3/4 rounded-lg" />
						<Skeleton class="mt-2 h-3 w-1/2 rounded-lg" />
					</div>
				</div>
			{/each}
		</div>
	{:else if suggestionsQuery.data}
		<div class="flex flex-col gap-2.5">
			{#each suggestionsQuery.data.pages as page}
				{#each (page.videos || []).filter((v: any) => !v.isShort) as nextVideo}
					<a
						href={`/watch/${nextVideo.fileId || nextVideo.id}`}
						class="group hover:bg-secondary/50 -mx-1.5 flex items-start gap-3 rounded-2xl p-2 transition-colors"
					>
						<div
							class="relative aspect-video w-36 sm:w-40 shrink-0 overflow-hidden rounded-xl border border-border/60 bg-card shadow-xs group-hover:border-primary/40 transition-colors"
						>
							<VideoThumbnail video={nextVideo} class="h-full w-full">
								<WatchLaterButton videoId={nextVideo.id} />
							</VideoThumbnail>
						</div>
						<div class="flex flex-1 flex-col py-0.5 min-w-0 overflow-hidden">
							<h4
								class="line-clamp-2 text-xs sm:text-sm leading-snug font-semibold text-foreground transition-colors group-hover:text-primary break-words [overflow-wrap:anywhere]"
								title={nextVideo.title}
							>
								{nextVideo.title}
							</h4>
							<span class="text-muted-foreground mt-1 text-[11px] font-medium">
								{nextVideo.uploader?.username || 'Unknown'}
							</span>
							<span class="text-muted-silver mt-0.5 text-[10px]">
								{nextVideo.views || 0} views • {formatTimeAgo(
									nextVideo.createdAt || nextVideo.created_at || new Date().toISOString()
								)}
							</span>
						</div>
					</a>
				{/each}
			{/each}

			{#if suggestionsQuery.hasNextPage}
				<div use:infiniteScroll class="flex justify-center py-4">
					{#if suggestionsQuery.isFetchingNextPage}
						<Loader2 class="text-primary h-5 w-5 animate-spin" />
					{/if}
				</div>
			{/if}
		</div>
	{:else}
		<div class="border-border/60 flex gap-2 rounded-2xl border-2 border-dashed p-6 text-center">
			<span class="text-muted-foreground text-xs sm:text-sm">No recommendations found.</span>
		</div>
	{/if}
</div>
