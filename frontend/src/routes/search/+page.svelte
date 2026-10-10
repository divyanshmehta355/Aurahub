<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { fetchApi } from '#lib/api';
	import VideoThumbnail from '#lib/components/VideoThumbnail.svelte';
	import WatchLaterButton from '#lib/components/WatchLaterButton.svelte';
	import { Search as SearchIcon, ShieldAlert } from 'lucide-svelte';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { Skeleton } from '#lib/components/ui/skeleton';
	import { Button } from '#lib/components/ui/button';

	const queryClient = useQueryClient();

	let queryStr = $derived(page.url.searchParams.get('q') || '');
	let adultParam = $derived(page.url.searchParams.get('adult'));

	let includeAdult = $state(false);

	$effect(() => {
		if (typeof window !== 'undefined') {
			if (adultParam !== null) {
				includeAdult = adultParam === 'true';
			} else {
				includeAdult = localStorage.getItem('showAdultContent') === 'true';
			}
		}
	});

	function toggleAdult(enable: boolean) {
		includeAdult = enable;
		const url = new URL(page.url.href);
		if (enable) {
			url.searchParams.set('adult', 'true');
		} else {
			url.searchParams.delete('adult');
		}
		goto(url.toString(), { replaceState: true });
	}

	const searchQuery = createQuery(
		() => ({
			queryKey: ['search', queryStr, includeAdult],
			queryFn: async () => {
				if (!queryStr) return { videos: [], total: 0, adultHidden: false, adultCount: 0 };
				const adultSuffix = includeAdult ? '&adult=true' : '';
				const res = await fetchApi(`/videos/search?q=${encodeURIComponent(queryStr)}${adultSuffix}`);
				return {
					videos: res.videos || res.data || [],
					total: res.total || (res.videos || []).length,
					adultHidden: !!res.adultHidden,
					adultCount: res.adultCount || 0
				};
			}
		}),
		() => queryClient
	);

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
	<title>{queryStr ? `${queryStr} - Search` : 'Search'} - Aurahub</title>
</svelte:head>

<div class="space-y-4 px-4 md:space-y-6 md:px-0">
	<div class="flex flex-wrap items-center justify-between gap-3 border-b pt-2 pb-4 md:pt-0">
		<h1 class="text-xl font-bold tracking-tight md:text-2xl">
			{#if queryStr}
				Results for "{queryStr}"
			{:else}
				Search Videos
			{/if}
		</h1>

		<!-- 18+ Content Filter Toggle -->
		<label class="hover:bg-muted border-border/80 bg-muted/40 flex cursor-pointer items-center gap-2 rounded-full border px-3.5 py-1.5 text-xs font-medium transition-colors select-none">
			<input
				type="checkbox"
				checked={includeAdult}
				onchange={(e) => toggleAdult(e.currentTarget.checked)}
				class="accent-primary h-3.5 w-3.5 rounded"
			/>
			<span>Include 18+ content</span>
		</label>
	</div>

	{#if searchQuery.isPending && queryStr}
		<div class="flex max-w-4xl flex-col gap-4">
			{#each Array(5) as _}
				<div class="flex animate-pulse flex-col gap-4 sm:flex-row">
					<Skeleton class="aspect-video w-full shrink-0 rounded-xl sm:w-80" />
					<div class="flex-1 space-y-3 py-2">
						<Skeleton class="h-6 w-3/4 rounded" />
						<Skeleton class="h-4 w-1/4 rounded" />
						<div class="mt-4 flex items-center gap-3">
							<Skeleton class="h-8 w-8 rounded-full" />
							<Skeleton class="h-4 w-1/3 rounded" />
						</div>
					</div>
				</div>
			{/each}
		</div>
	{:else if searchQuery.isError}
		<div class="flex flex-col items-center justify-center py-12 text-center">
			<div class="bg-destructive/10 text-destructive rounded-xl p-4">
				<p>{searchQuery.error.message || 'Failed to search videos.'}</p>
				<Button variant="outline" class="mt-4" onclick={() => searchQuery.refetch()}
					>Try again</Button
				>
			</div>
		</div>
	{:else if searchQuery.data?.adultHidden && searchQuery.data.videos.length === 0}
		<!-- 18+ Content Hidden Banner (Zero safe results) -->
		<div class="border-amber-500/30 bg-amber-500/10 mx-4 flex flex-col items-center justify-center rounded-2xl border p-8 text-center md:mx-0">
			<ShieldAlert class="text-amber-400 mb-3 h-10 w-10" />
			<h2 class="text-lg font-semibold text-foreground">18+ Content Filtered</h2>
			<p class="text-muted-foreground mt-1.5 max-w-md text-sm">
				No non-adult videos matched "{queryStr}", but <strong>{searchQuery.data.adultCount}</strong> 18+ video(s) are available.
			</p>
			<Button
				variant="default"
				size="sm"
				class="mt-4"
				onclick={() => toggleAdult(true)}
			>
				Show 18+ Results ({searchQuery.data.adultCount})
			</Button>
		</div>
	{:else if !queryStr || searchQuery.data?.videos.length === 0}
		<div
			class="border-muted-foreground/20 mx-4 flex flex-col items-center justify-center rounded-2xl border-2 border-dashed py-16 text-center md:mx-0 md:py-24"
		>
			<SearchIcon class="text-muted-foreground mb-4 h-12 w-12 opacity-50" />
			<h2 class="mb-2 text-lg font-semibold md:text-xl">No results found</h2>
			<p class="text-muted-foreground max-w-sm px-4 text-sm md:text-base">
				{#if queryStr}
					Try different keywords or check spelling.
				{:else}
					Enter a search term above to find videos.
				{/if}
			</p>
		</div>
	{:else if searchQuery.data}
		<div class="flex max-w-4xl flex-col gap-6 md:gap-4">
			<!-- Subtle note if adult videos were also filtered out -->
			{#if searchQuery.data.adultHidden && !includeAdult}
				<div class="border-amber-500/30 bg-amber-500/10 text-amber-300 flex items-center justify-between rounded-xl border px-4 py-2.5 text-xs">
					<span class="flex items-center gap-2">
						<ShieldAlert class="h-4 w-4 shrink-0 text-amber-400" />
						{searchQuery.data.adultCount} adult video(s) hidden by safe search filter.
					</span>
					<button
						type="button"
						class="text-primary hover:underline font-semibold"
						onclick={() => toggleAdult(true)}
					>
						Show 18+
					</button>
				</div>
			{/if}

			{#each searchQuery.data.videos.filter((v: any) => !v.isShort) as video}
				<a
					href={`/watch/${video.fileId || video.id}`}
					class="hover:bg-muted/50 group flex flex-col gap-3 rounded-xl transition-colors sm:flex-row sm:p-2 md:gap-4"
				>
					<!-- Thumbnail -->
					<div class="w-full shrink-0 sm:w-80">
						<VideoThumbnail
							{video}
							class="aspect-video rounded-xl sm:rounded-lg"
							imgClass="transition-transform duration-300 group-hover:scale-105"
						>
							<WatchLaterButton videoId={video.id} />
						</VideoThumbnail>
					</div>
					<!-- Metadata -->
					<div class="flex flex-1 flex-col px-1 py-1 sm:px-0">
						<h3
							class="line-clamp-2 text-base leading-tight font-semibold transition-colors group-hover:text-primary md:text-lg mb-1"
						>
							{video.title}
						</h3>
						<div class="text-muted-foreground mb-3 text-[11px] md:text-xs lg:text-sm">
							<span>{video.views || 0} views</span>
							<span class="mx-1">•</span>
							<span>{formatTimeAgo(video.created_at || video.createdAt)}</span>
						</div>

						<div class="mb-2 flex items-center gap-2 md:mb-3 md:gap-3">
							<img
								src={video.uploader_avatar ||
									video.uploader?.avatar ||
									`https://api.dicebear.com/7.x/identicon/svg?seed=${video.uploader_username || video.uploader?.username}`}
								alt={video.uploader_username || video.uploader?.username}
								class="bg-muted h-6 w-6 rounded-full object-cover md:h-8 md:w-8"
							/>
							<p class="text-[13px] font-medium transition-colors hover:text-primary md:text-sm">
								{video.uploader_username || video.uploader?.username}
							</p>
						</div>

						{#if video.description}
							<p class="text-muted-foreground line-clamp-1 text-[11px] md:line-clamp-2 md:text-xs">
								{video.description}
							</p>
						{/if}
					</div>
				</a>
			{/each}
		</div>
	{/if}
</div>
