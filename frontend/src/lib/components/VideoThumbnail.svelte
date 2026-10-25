<script lang="ts">
	import { getFallbackThumbnailUrl } from '#lib/thumbnailSvg';
	import { fetchApi } from '#lib/api';

	let {
		video,
		class: className = '',
		imgClass = '',
		children = undefined
	}: {
		video: any;
		class?: string;
		imgClass?: string;
		children?: import('svelte').Snippet;
	} = $props();

	let fallbackUrl = $derived(
		getFallbackThumbnailUrl(video.fileId || video.id, video.title, video.category)
	);
	let fetchedUrl = $state('');
	let isFetching = $state(false);

	let imageUrl = $derived(video.thumbnailUrl || fetchedUrl || fallbackUrl);

	$effect(() => {
		const videoId = video.id || video.fileId;
		if (video.thumbnailUrl || !videoId) {
			fetchedUrl = '';
			isFetching = false;
			return;
		}

		let isActive = true;
		isFetching = true;
		fetchedUrl = '';

		fetchApi(`/videos/${videoId}/thumbnail`)
			.then((res) => {
				if (isActive && res && res.thumbnailUrl) {
					fetchedUrl = res.thumbnailUrl;
				}
			})
			.catch(() => {})
			.finally(() => {
				if (isActive) isFetching = false;
			});

		return () => {
			isActive = false;
		};
	});
	let isLoaded = $state(false);
</script>

<div class="relative overflow-hidden bg-slate-surface border border-slate-border/50 {className}">
	{#if isFetching}
		<div class="h-full w-full bg-slate-elevated/60 animate-pulse"></div>
	{:else}
		<img
			src={imageUrl}
			alt={video.title}
			loading="lazy"
			onload={() => (isLoaded = true)}
			class="h-full w-full object-cover transition-all duration-300 {isLoaded ? 'opacity-100' : 'opacity-80'} {imgClass}"
		/>
	{/if}

	{@render children?.()}
</div>
