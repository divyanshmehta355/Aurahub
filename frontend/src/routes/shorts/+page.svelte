<script lang="ts">
	import { onMount } from 'svelte';
	import { fetchApi } from '#lib/api';
	import { Play } from 'lucide-svelte';
	import confetti from 'canvas-confetti';
	import { createInfiniteQuery, useQueryClient } from '@tanstack/svelte-query';

	const queryClient = useQueryClient();

	let activeIndex = $state(0);
	let hasFiredConfetti = $state(false);
	let containerRef: HTMLDivElement;

	const shortsQuery = createInfiniteQuery(
		() => ({
			queryKey: ['shorts'],
			queryFn: async ({ pageParam = 1 }) => {
				const res = await fetchApi(`/videos?type=short&limit=10&page=${pageParam}`);
				return {
					videos: res.videos || res.data || [],
					currentPage: res.currentPage || pageParam,
					totalPages: res.totalPages || 1
				};
			},
			getNextPageParam: (lastPage) => {
				return lastPage.currentPage < lastPage.totalPages ? lastPage.currentPage + 1 : undefined;
			},
			initialPageParam: 1
		}),
		() => queryClient
	);

	// Derived flat array of videos from infinite query pages
	let shorts = $derived(shortsQuery.data?.pages.flatMap((p) => p.videos) || []);
	let hasMore = $derived(shortsQuery.hasNextPage);

	$effect(() => {
		if (
			shortsQuery.isSuccess &&
			!hasMore &&
			shorts.length > 0 &&
			activeIndex === shorts.length - 1 &&
			!hasFiredConfetti
		) {
			hasFiredConfetti = true;
			confetti({
				particleCount: 150,
				spread: 70,
				origin: { y: 0.6 },
				colors: ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6']
			});
		}
	});

	function handleScroll() {
		if (!containerRef) return;
		const { scrollTop, clientHeight, scrollHeight } = containerRef;

		const newIndex = Math.round(scrollTop / clientHeight);
		if (newIndex !== activeIndex) {
			activeIndex = newIndex;
		}

		if (
			hasMore &&
			!shortsQuery.isFetchingNextPage &&
			scrollTop + clientHeight >= scrollHeight - clientHeight * 2
		) {
			shortsQuery.fetchNextPage();
		}
	}

	function handleShare(id: string) {
		const url = `${window.location.origin}/watch/${id}`;
		navigator.clipboard.writeText(url);
		alert('Link copied to clipboard!');
	}
</script>

<svelte:head>
	<title>Shorts - Aurahub</title>
</svelte:head>

<div
	class="hide-scrollbar relative mx-auto h-[calc(100vh-[env(safe-area-inset-bottom)]-4rem)] w-full max-w-[450px] snap-y snap-mandatory overflow-x-hidden overflow-y-scroll bg-black md:h-[calc(100vh-4rem)] md:max-w-[500px]"
	bind:this={containerRef}
	onscroll={handleScroll}
>
	{#if shortsQuery.isPending}
		<div class="flex h-full w-full items-center justify-center">
			<div
				class="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent"
			></div>
		</div>
	{:else if shortsQuery.isError}
		<div
			class="text-destructive flex h-full w-full items-center justify-center bg-background p-4 text-center"
		>
			<p>{shortsQuery.error?.message || 'Failed to load shorts.'}</p>
		</div>
	{:else if shorts.length === 0}
		<div
			class="flex h-full w-full flex-col items-center justify-center bg-background p-4 text-center"
		>
			<Play class="text-muted-foreground/50 mb-4 h-16 w-16" />
			<h2 class="mb-2 text-xl font-bold">No Shorts found</h2>
			<p class="text-muted-foreground">Check back later for new short videos!</p>
		</div>
	{:else}
		{#each shorts as short, index}
			<div
				class="relative flex h-full w-full snap-start snap-always items-center justify-center bg-black"
			>
				<iframe
					src={`https://streamtape.com/e/${short.fileId || short.id}`}
					title={short.title}
					class="h-full w-full object-cover"
					frameborder="0"
					allowfullscreen
					allow="autoplay"
				></iframe>

				<!-- Overlay UI -->
				<div
					class="pointer-events-none absolute inset-x-0 bottom-0 flex items-end justify-between bg-gradient-to-t from-black/80 via-black/40 to-transparent p-4 pb-20 md:pb-4"
				>
					<div class="pointer-events-auto flex-1 pr-12">
						<a href={`/profile/${short.uploader?.username}`} class="mb-3 flex items-center gap-2">
							<img
								src={short.uploader?.avatar ||
									`https://api.dicebear.com/7.x/identicon/svg?seed=${short.uploader?.username}`}
								alt="Avatar"
								class="h-8 w-8 rounded-full border border-white/20 shadow-sm"
							/>
							<span class="text-[13px] font-semibold text-white drop-shadow-md md:text-sm"
								>@{short.uploader?.username}</span
							>
						</a>
						<a href={`/watch/${short.fileId || short.id}`}>
							<h3
								class="mb-1 line-clamp-2 text-sm font-medium text-white drop-shadow-md md:text-base"
							>
								{short.title}
							</h3>
						</a>
					</div>
				</div>

				<!-- Right Actions Sidebar -->
				<div
					class="pointer-events-auto absolute right-3 bottom-24 flex flex-col items-center gap-5 md:right-4 md:bottom-20 md:gap-6"
				>
					<button
						class="group flex flex-col items-center gap-1 transition-transform hover:scale-110 active:scale-95"
						aria-label="Like"
					>
						<div
							class="flex h-10 w-10 items-center justify-center rounded-full bg-black/40 backdrop-blur-md transition-colors group-hover:bg-black/60 md:h-12 md:w-12"
						>
							<svg
								xmlns="http://www.w3.org/2000/svg"
								width="24"
								height="24"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="2"
								stroke-linecap="round"
								stroke-linejoin="round"
								class="h-5 w-5 text-white md:h-6 md:w-6"
								><path d="M7 10v12" /><path
									d="M15 5.88 14 10h5.83a2 2 0 0 1 1.92 2.56l-2.33 8A2 2 0 0 1 17.5 22H4a2 2 0 0 1-2-2v-8a2 2 0 0 1 2-2h2.76a2 2 0 0 0 1.79-1.11L12 2a3.13 3.13 0 0 1 3 3.88Z"
								/></svg
							>
						</div>
						<span class="text-xs font-semibold text-white drop-shadow-md"
							>{short.likesCount || 0}</span
						>
					</button>

					<button
						class="group flex flex-col items-center gap-1 transition-transform hover:scale-110 active:scale-95"
						aria-label="Comment"
					>
						<div
							class="flex h-10 w-10 items-center justify-center rounded-full bg-black/40 backdrop-blur-md transition-colors group-hover:bg-black/60 md:h-12 md:w-12"
						>
							<svg
								xmlns="http://www.w3.org/2000/svg"
								width="24"
								height="24"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="2"
								stroke-linecap="round"
								stroke-linejoin="round"
								class="h-5 w-5 text-white md:h-6 md:w-6"
								><path d="M7.9 20A9 9 0 1 0 4 16.1L2 22Z" /></svg
							>
						</div>
						<span class="text-xs font-semibold text-white drop-shadow-md">0</span>
					</button>

					<button
						class="group flex flex-col items-center gap-1 transition-transform hover:scale-110 active:scale-95"
						onclick={() => handleShare(short.fileId || short.id)}
						aria-label="Share"
					>
						<div
							class="flex h-10 w-10 items-center justify-center rounded-full bg-black/40 backdrop-blur-md transition-colors group-hover:bg-black/60 md:h-12 md:w-12"
						>
							<svg
								xmlns="http://www.w3.org/2000/svg"
								width="24"
								height="24"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="2"
								stroke-linecap="round"
								stroke-linejoin="round"
								class="h-5 w-5 text-white md:h-6 md:w-6"
								><circle cx="18" cy="5" r="3" /><circle cx="6" cy="12" r="3" /><circle
									cx="18"
									cy="19"
									r="3"
								/><line x1="8.59" x2="15.42" y1="13.51" y2="17.49" /><line
									x1="15.41"
									x2="8.59"
									y1="6.51"
									y2="10.49"
								/></svg
							>
						</div>
						<span class="text-xs font-semibold text-white drop-shadow-md">Share</span>
					</button>
				</div>
			</div>
		{/each}

		{#if shortsQuery.isFetchingNextPage}
			<div class="flex h-[200px] w-full snap-start items-center justify-center bg-black">
				<div
					class="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent"
				></div>
			</div>
		{/if}
	{/if}
</div>

<style>
	.hide-scrollbar::-webkit-scrollbar {
		display: none;
	}
	.hide-scrollbar {
		-ms-overflow-style: none;
		scrollbar-width: none;
	}
</style>
