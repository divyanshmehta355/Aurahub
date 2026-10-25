<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { fetchApi } from '#lib/api';
	import { contentModeState } from '#lib/consent.svelte';
	import { getFallbackThumbnailUrl } from '#lib/thumbnailSvg';
	import VideoThumbnail from '#lib/components/VideoThumbnail.svelte';
	import WatchLaterButton from '#lib/components/WatchLaterButton.svelte';
	import { Clock, Play, RotateCw } from 'lucide-svelte';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { Skeleton } from '#lib/components/ui/skeleton';
	import { Button } from '#lib/components/ui/button';

	const queryClient = useQueryClient();

	let currentPage = $derived(Number(page.url.searchParams.get('page')) || 1);
	const limit = 12;
	function generateSeed(): string {
		return Math.random().toString(36).substring(2, 10);
	}

	let feedSeed = $state<string>(generateSeed());

	function refreshFeed() {
		feedSeed = generateSeed();
		queryClient.invalidateQueries({ queryKey: ['videos'] });
		if (currentPage !== 1) {
			goToPage(1);
		}
	}

	let query = createQuery(
		() => ({
			queryKey: ['videos', currentPage, limit, contentModeState.isAdult, feedSeed],
			queryFn: async () => {
				const seedParam = feedSeed ? `&seed=${feedSeed}` : '';
				const res = await fetchApi(`/videos?type=standard&page=${currentPage}&limit=${limit}&sort=recommended${seedParam}`);
				return res;
			}
		}),
		() => queryClient
	);

	function goToPage(p: number) {
		const totalPages = query.data?.totalPages || 1;
		if (p < 1 || p > totalPages) return;
		const url = new URL(page.url.toString());
		url.searchParams.set('page', p.toString());
		goto(url.toString());
	}

	function getVisiblePages(current: number, total: number) {
		let start = Math.max(1, current - 2);
		let end = Math.min(total, current + 2);

		if (current <= 3) {
			end = Math.min(total, 5);
		}
		if (current >= total - 2) {
			start = Math.max(1, total - 4);
		}

		const pages = [];
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

	function formatDuration(seconds: number) {
		if (!seconds) return '0:00';
		const m = Math.floor(seconds / 60);
		const s = Math.floor(seconds % 60);
		return `${m}:${s.toString().padStart(2, '0')}`;
	}
	let standardVideos = $derived((query.data?.videos || []).filter((v: any) => !v.isShort));
	let featuredVideo = $derived(currentPage === 1 && standardVideos.length > 0 ? standardVideos[0] : null);
	let gridVideos = $derived(currentPage === 1 && standardVideos.length > 0 ? standardVideos.slice(1) : standardVideos);
</script>

<svelte:head>
	<title>{contentModeState.isAdult ? 'Aurahub - 18+ Adult Feed' : 'Aurahub - Stream Cinema & Creators'}</title>
</svelte:head>

<div class="space-y-8 pb-10">
	{#if featuredVideo && !query.isPending}
		<!-- Cinematic Featured Spotlight Hero -->
		<section class="group relative overflow-hidden rounded-3xl border border-border/70 bg-card shadow-2xl transition-all duration-300">
			<a href={`/watch/${featuredVideo.fileId || featuredVideo.id}`} class="block relative aspect-[16/9] sm:aspect-[21/9] w-full overflow-hidden">
				<VideoThumbnail
					video={featuredVideo}
					class="h-full w-full"
					imgClass="h-full w-full object-cover transition-transform duration-700 ease-out group-hover:scale-105"
				/>

				<!-- Vignette & Cinematic Backlight Overlays -->
				<div class="absolute inset-0 bg-gradient-to-t from-deep-slate via-deep-slate/50 to-transparent"></div>
				<div class="absolute inset-0 bg-gradient-to-r from-deep-slate/90 via-deep-slate/40 to-transparent"></div>

				<!-- Hero Content Details -->
				<div class="absolute inset-x-0 bottom-0 p-6 sm:p-8 lg:p-10 flex flex-col justify-end">
					<div class="flex items-center gap-2 mb-3">
						<span class="inline-flex items-center gap-1.5 rounded-full border border-primary/40 bg-primary/20 px-3 py-1 text-[11px] font-bold uppercase tracking-wider text-primary backdrop-blur-md">
							<span class="h-1.5 w-1.5 rounded-full bg-primary animate-pulse"></span>
							Spotlight Selection
						</span>
						{#if featuredVideo.isAdult}
							<span class="rounded-full bg-rose-500/20 border border-rose-500/40 px-2.5 py-0.5 text-xs font-bold text-rose-400">18+</span>
						{/if}
						{#if featuredVideo.category}
							<span class="rounded-full bg-secondary/80 border border-border/60 px-2.5 py-0.5 text-xs font-medium text-muted-foreground backdrop-blur-sm">
								{featuredVideo.category}
							</span>
						{/if}
					</div>

					<h2 class="font-serif text-2xl sm:text-3xl lg:text-4xl font-bold tracking-tight text-white mb-2 max-w-3xl line-clamp-2">
						{featuredVideo.title}
					</h2>

					{#if featuredVideo.description}
						<p class="text-xs sm:text-sm text-zinc-300/90 max-w-2xl line-clamp-2 mb-4 leading-relaxed hidden sm:block">
							{featuredVideo.description}
						</p>
					{/if}

					<div class="flex flex-wrap items-center gap-4 pt-1">
						<div class="inline-flex items-center gap-2 rounded-full bg-primary px-5 py-2 text-xs sm:text-sm font-semibold text-primary-foreground shadow-lg shadow-primary/25 transition-transform group-hover:scale-105">
							<Play class="h-4 w-4 fill-current" />
							<span>Watch Premiere</span>
						</div>

						<div class="flex items-center gap-2 text-xs text-zinc-300">
							<img
								src={featuredVideo.uploader_avatar || `https://api.dicebear.com/7.x/identicon/svg?seed=${featuredVideo.uploader_username}`}
								alt={featuredVideo.uploader_username}
								class="h-6 w-6 rounded-full border border-white/20 object-cover"
							/>
							<span class="font-medium text-white">@{featuredVideo.uploader_username}</span>
							<span>•</span>
							<span>{featuredVideo.views || 0} views</span>
							<span>•</span>
							<span>{formatTimeAgo(featuredVideo.createdAt || featuredVideo.created_at)}</span>
						</div>
					</div>
				</div>
			</a>
		</section>
	{/if}

	<!-- Feed Section Header -->
	<div class="flex items-center justify-between border-b border-border/50 pb-4">
		<div class="flex items-center gap-2.5">
			<h1 class="text-xl sm:text-2xl font-bold tracking-tight text-foreground font-sans">
				{contentModeState.isAdult ? '18+ Adult Cinema' : 'Curated Stream'}
			</h1>
			{#if contentModeState.isAdult}
				<span class="rounded-full bg-rose-500/15 border border-rose-500/30 px-2.5 py-0.5 text-xs font-bold text-rose-400">18+</span>
			{/if}
		</div>

		<Button
			variant="outline"
			size="sm"
			class="h-8 gap-1.5 rounded-full border-border/80 text-xs text-muted-foreground hover:text-foreground hover:border-primary/40"
			onclick={refreshFeed}
			title="Shuffle and explore fresh recommendations"
		>
			<RotateCw class="h-3.5 w-3.5 {query.isFetching ? 'animate-spin' : ''}" />
			<span class="hidden sm:inline">Shuffle Feed</span>
		</Button>
	</div>

	{#if query.isPending}
		<div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
			{#each Array(8) as _}
				<div class="flex animate-pulse flex-col space-y-3">
					<Skeleton class="aspect-video w-full rounded-2xl" />
					<div class="flex gap-3">
						<Skeleton class="h-10 w-10 shrink-0 rounded-full" />
						<div class="w-full space-y-2">
							<Skeleton class="h-4 w-3/4 rounded" />
							<Skeleton class="h-3 w-1/2 rounded" />
						</div>
					</div>
				</div>
			{/each}
		</div>
	{:else if query.isError}
		<div class="flex flex-col items-center justify-center py-16 text-center">
			<div class="bg-destructive/10 border border-destructive/20 text-destructive rounded-2xl p-6 max-w-md">
				<p class="font-medium text-sm">{query.error.message}</p>
				<Button variant="outline" class="mt-4 rounded-full" onclick={() => query.refetch()}>Try again</Button>
			</div>
		</div>
	{:else if standardVideos.length === 0}
		<div
			class="border-border/60 bg-card/40 flex flex-col items-center justify-center rounded-3xl border-2 border-dashed py-24 text-center px-4"
		>
			<div class="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-secondary border border-border/80 text-primary">
				<Play class="h-8 w-8 opacity-75" />
			</div>
			<h2 class="mb-2 text-xl font-bold text-foreground">No videos available</h2>
			<p class="text-muted-foreground max-w-sm text-xs sm:text-sm">
				When creators publish new releases, they will appear in your curated feed.
			</p>
		</div>
	{:else}
		<!-- Video Cards Grid -->
		<div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
			{#each gridVideos as video}
				<a href={`/watch/${video.fileId || video.id}`} class="group flex flex-col space-y-3 min-w-0 transition-transform">
					<!-- Thumbnail Container -->
					<div class="relative overflow-hidden rounded-2xl border border-border/60 bg-card shadow-sm group-hover:border-primary/40 group-hover:shadow-md transition-all">
						<VideoThumbnail
							{video}
							class="aspect-video w-full rounded-2xl"
							imgClass="transition-transform duration-500 ease-out group-hover:scale-105"
						>
							<WatchLaterButton videoId={video.id} />
							{#if video.duration}
								<div class="absolute bottom-2 right-2 rounded-md bg-black/80 px-2 py-0.5 text-[11px] font-semibold text-white backdrop-blur-md border border-white/10">
									{formatDuration(video.duration)}
								</div>
							{/if}
						</VideoThumbnail>
					</div>

					<!-- Metadata -->
					<div class="flex gap-3 min-w-0 px-0.5">
						<img
							src={video.uploader_avatar ||
								`https://api.dicebear.com/7.x/identicon/svg?seed=${video.uploader_username}`}
							alt={video.uploader_username}
							class="h-9 w-9 shrink-0 rounded-full border border-border/80 object-cover mt-0.5"
						/>
						<div class="flex flex-col overflow-hidden min-w-0 flex-1">
							<h3
								class="line-clamp-2 text-sm leading-snug font-semibold text-foreground transition-colors group-hover:text-primary break-words [overflow-wrap:anywhere]"
								title={video.title}
							>
								{video.title}
							</h3>
							<div class="mt-1 space-y-0.5 text-xs text-muted-foreground">
								<p class="transition-colors hover:text-foreground font-medium">{video.uploader_username}</p>
								<div class="flex items-center gap-1.5 text-[11px] text-muted-silver">
									<span>{video.views || 0} views</span>
									<span>•</span>
									<span>{formatTimeAgo(video.createdAt || video.created_at)}</span>
								</div>
							</div>
						</div>
					</div>
				</a>
			{/each}
		</div>

		<!-- Pagination Controls -->
		{#if query.data?.totalPages > 1}
			<div class="mt-12 flex justify-center pb-8">
				<nav class="flex items-center gap-2">
					<Button
						variant="outline"
						size="icon"
						disabled={currentPage === 1}
						onclick={() => goToPage(1)}
						class="hidden sm:flex rounded-full border-border/70"
					>
						«
					</Button>
					<Button
						variant="secondary"
						disabled={currentPage === 1}
						onclick={() => goToPage(currentPage - 1)}
						class="rounded-full px-4 text-xs font-semibold"
					>
						Previous
					</Button>

					<div class="mx-2 flex items-center gap-1">
						{#each getVisiblePages(currentPage, query.data.totalPages) as p}
							<Button
								variant={currentPage === p ? 'default' : 'ghost'}
								size="icon"
								onclick={() => goToPage(p)}
								class="rounded-full h-8 w-8 text-xs font-semibold"
							>
								{p}
							</Button>
						{/each}
					</div>

					<Button
						variant="secondary"
						disabled={currentPage === query.data.totalPages}
						onclick={() => goToPage(currentPage + 1)}
						class="rounded-full px-4 text-xs font-semibold"
					>
						Next
					</Button>
					<Button
						variant="outline"
						size="icon"
						disabled={currentPage === query.data.totalPages}
						onclick={() => goToPage(query.data.totalPages)}
						class="hidden sm:flex rounded-full border-border/70"
					>
						»
					</Button>
				</nav>
			</div>
		{/if}
	{/if}
</div>
