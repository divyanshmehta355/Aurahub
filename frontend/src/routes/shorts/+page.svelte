<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { fetchApi } from '#lib/api';
	import { ThumbsUp, MessageSquare, Share2, Play, PartyPopper } from 'lucide-svelte';
	import confetti from 'canvas-confetti';

	let shorts: any[] = $state([]);
	let isLoading = $state(true);
	let error = $state('');
	let activeIndex = $state(0);
	
	let page = $state(1);
	let hasMore = $state(true);
	let isLoadingMore = $state(false);
	let hasFiredConfetti = $state(false);
	
	let containerRef: HTMLDivElement;

	$effect(() => {
		if (!hasMore && shorts.length > 0 && activeIndex === shorts.length && !hasFiredConfetti) {
			hasFiredConfetti = true;
			confetti({
				particleCount: 150,
				spread: 70,
				origin: { y: 0.6 },
				colors: ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6']
			});
		}
	});

	async function loadShorts(pageNum: number) {
		try {
			const res = await fetchApi(`/videos?type=short&limit=10&page=${pageNum}`);
			const newShorts = res.videos || res.data || [];
			if (newShorts.length === 0) {
				hasMore = false;
			} else {
				if (pageNum === 1) {
					shorts = newShorts;
				} else {
					shorts = [...shorts, ...newShorts];
				}
				if (newShorts.length < 10) {
					hasMore = false;
				}
			}
		} catch (err: any) {
			if (pageNum === 1) error = err.message || 'Failed to load shorts.';
		}
	}

	onMount(async () => {
		await loadShorts(1);
		isLoading = false;
	});

	function handleScroll() {
		if (!containerRef) return;
		const { scrollTop, clientHeight, scrollHeight } = containerRef;
		
		const newIndex = Math.round(scrollTop / clientHeight);
		if (newIndex !== activeIndex) {
			activeIndex = newIndex;
		}

		if (hasMore && !isLoadingMore && (scrollTop + clientHeight) >= (scrollHeight - clientHeight * 2)) {
			loadMore();
		}
	}

	async function loadMore() {
		isLoadingMore = true;
		page += 1;
		await loadShorts(page);
		isLoadingMore = false;
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
	class="h-[calc(100vh-4rem)] w-full max-w-[500px] mx-auto bg-black relative snap-y snap-mandatory overflow-y-scroll overflow-x-hidden hide-scrollbar"
	bind:this={containerRef}
	onscroll={handleScroll}
>
	{#if isLoading}
		<div class="h-full w-full flex items-center justify-center">
			<div class="w-8 h-8 border-4 border-primary border-t-transparent rounded-full animate-spin"></div>
		</div>
	{:else if error}
		<div class="h-full w-full flex items-center justify-center p-4 text-center text-red-500 bg-background">
			<p>{error}</p>
		</div>
	{:else if shorts.length === 0}
		<div class="h-full w-full flex flex-col items-center justify-center p-4 text-center bg-background">
			<Play class="w-16 h-16 text-muted-foreground/50 mb-4" />
			<h2 class="text-xl font-bold mb-2">No Shorts found</h2>
			<p class="text-muted-foreground">Check back later for new short videos!</p>
		</div>
	{:else}
		{#each shorts as short, index}
			<div class="h-full w-full snap-start snap-always relative flex items-center justify-center bg-black">
				<iframe 
					src={`https://streamtape.com/e/${short.fileId || short.id}`} 
					title={short.title}
					class="w-full h-full object-cover"
					frameborder="0"
					allowfullscreen
					allow="autoplay"
				></iframe>

				<!-- Overlay UI -->
				<div class="absolute bottom-0 inset-x-0 p-4 bg-gradient-to-t from-black/80 via-black/40 to-transparent pointer-events-none flex items-end justify-between">
					<div class="flex-1 pr-12 pointer-events-auto">
						<a href={`/profile/${short.uploader?.username}`} class="flex items-center gap-2 mb-3">
							<img src={short.uploader?.avatar || `https://api.dicebear.com/7.x/identicon/svg?seed=${short.uploader?.username}`} alt="Avatar" class="w-8 h-8 rounded-full border border-white/20" />
							<span class="text-white font-semibold text-sm">@{short.uploader?.username}</span>
						</a>
						<a href={`/watch/${short.fileId || short.id}`}>
							<h3 class="text-white font-medium text-sm line-clamp-2 mb-1">{short.title}</h3>
						</a>
					</div>
				</div>

				<!-- Right Actions Sidebar -->
				<div class="absolute right-4 bottom-20 flex flex-col gap-6 items-center pointer-events-auto">
					<button class="flex flex-col items-center gap-1 group">
						<div class="w-12 h-12 rounded-full bg-black/40 flex items-center justify-center text-white group-hover:bg-black/60 transition-colors">
							<ThumbsUp class="w-6 h-6" />
						</div>
						<span class="text-white text-xs font-semibold">{short.likesCount || 0}</span>
					</button>
					<a href={`/watch/${short.fileId || short.id}`} class="flex flex-col items-center gap-1 group">
						<div class="w-12 h-12 rounded-full bg-black/40 flex items-center justify-center text-white group-hover:bg-black/60 transition-colors">
							<MessageSquare class="w-6 h-6" />
						</div>
						<span class="text-white text-xs font-semibold">{short.commentCount || 0}</span>
					</a>
					<button onclick={() => handleShare(short.fileId || short.id)} class="flex flex-col items-center gap-1 group">
						<div class="w-12 h-12 rounded-full bg-black/40 flex items-center justify-center text-white group-hover:bg-black/60 transition-colors">
							<Share2 class="w-6 h-6" />
						</div>
						<span class="text-white text-xs font-semibold">Share</span>
					</button>
				</div>
			</div>
		{/each}
		{#if !hasMore && shorts.length > 0}
			<div class="h-full w-full snap-start snap-always relative flex flex-col items-center justify-center bg-black text-center p-8">
				<div class="w-20 h-20 bg-primary/20 rounded-full flex items-center justify-center mb-6">
					<PartyPopper class="w-10 h-10 text-primary" />
				</div>
				<h2 class="text-3xl font-bold text-white mb-3">You've reached the end!</h2>
				<p class="text-muted-foreground text-lg mb-8">You've watched all available Shorts. Check back later for more!</p>
				<a href="/" class="px-6 py-3 bg-primary text-primary-foreground font-semibold rounded-full hover:bg-primary/90 transition-colors">
					Back to Home
				</a>
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
