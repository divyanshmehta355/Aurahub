<script lang="ts">
	import { onMount } from 'svelte';
	import { getFallbackThumbnailUrl } from '#lib/thumbnailSvg';
	import { fetchApi } from '#lib/api';

	let { video, class: className = '', imgClass = '', children } = $props();

	let fallbackUrl = $derived(getFallbackThumbnailUrl(video.fileId || video.id, video.title, video.category));
	let imageUrl = $state(video.thumbnailUrl || fallbackUrl);
	let isLoading = $state(!video.thumbnailUrl && !!(video.fileId || video.id));

	onMount(async () => {
		if (video.thumbnailUrl) {
			imageUrl = video.thumbnailUrl;
			isLoading = false;
			return;
		}

		if (!video.id && !video.fileId) {
			isLoading = false;
			return;
		}

		try {
			const videoId = video.id || video.fileId;
			const res = await fetchApi(`/videos/${videoId}/thumbnail`);
			if (res && res.thumbnailUrl) {
				imageUrl = res.thumbnailUrl;
			}
		} catch (err) {
			// keep fallback
		} finally {
			isLoading = false;
		}
	});
</script>

<div class="relative overflow-hidden bg-muted {className}">
	{#if isLoading}
		<div class="animate-pulse bg-muted h-full w-full"></div>
	{:else}
		<img 
			src={imageUrl} 
			alt={video.title}
			class="h-full w-full object-cover {imgClass}"
		/>
	{/if}
	
	{@render children?.()}
</div>
