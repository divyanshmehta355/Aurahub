<script lang="ts">
	import { getFallbackThumbnailUrl } from '#lib/thumbnailSvg';
	import { fetchApi } from '#lib/api';

	let { video, class: className = '', imgClass = '', children } = $props();

	let fallbackUrl = $derived(getFallbackThumbnailUrl(video.fileId || video.id, video.title, video.category));
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
			.then(res => {
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
</script>

<div class="relative overflow-hidden bg-muted {className}">
	{#if isFetching}
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
