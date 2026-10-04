<script lang="ts">
	import { onMount } from 'svelte';
	import { fetchApi } from '#lib/api';
	import { UploadCloud, FileVideo, X, CheckCircle, Link as LinkIcon, ListVideo } from 'lucide-svelte';
	import PlaylistUploadForm from '#lib/components/PlaylistUploadForm.svelte';

	let uploadType = $state('direct'); // 'direct', 'remote', 'playlist'
	let playlists: any[] = $state([]);

	// Common form
	let title = $state('');
	let description = $state('');
	let category = $state('Other');
	let isShort = $state(false);
	
	// Direct
	let file: File | null = $state(null);
	
	// Remote
	let videoUrl = $state('');

	let isUploading = $state(false);
	let uploadProgress = $state(0);
	let error = $state('');
	let success = $state(false);
	let fileId = $state('');
	let statusMessage = $state('');

	const CHUNK_SIZE = 5 * 1024 * 1024; // 5MB

	onMount(async () => {
		try {
			const res = await fetchApi('/playlists/my-playlists');
			playlists = res || [];
		} catch (err) {
			console.error('Could not fetch playlists', err);
		}
	});

	function handleFileSelect(e: Event) {
		const target = e.target as HTMLInputElement;
		if (target.files && target.files.length > 0) {
			file = target.files[0];
			title = file.name.replace(/\.[^/.]+$/, "");
			error = '';
		}
	}

	async function uploadDirect(e: Event) {
		e.preventDefault();
		if (!file) {
			error = 'Please select a video file.';
			return;
		}

		isUploading = true;
		error = '';
		uploadProgress = 0;
		statusMessage = 'Uploading chunks...';

		try {
			const initRes = await fetchApi('/videos/get-upload-url');
			const serverFileId = initRes.fileId;
			const totalChunks = Math.ceil(file.size / CHUNK_SIZE);
			
			let finalVideoId = '';
			for (let chunkIndex = 0; chunkIndex < totalChunks; chunkIndex++) {
				const start = chunkIndex * CHUNK_SIZE;
				const end = Math.min(start + CHUNK_SIZE, file.size);
				const chunk = file.slice(start, end);

				const formData = new FormData();
				formData.append('uploadId', serverFileId);
				formData.append('chunkIndex', chunkIndex.toString());
				formData.append('totalChunks', totalChunks.toString());
				formData.append('fileName', file.name);
				formData.append('chunk', chunk);

				const res = await fetchApi('/videos/upload-chunk', { method: 'POST', body: formData });
				if (res && res.completed) {
					finalVideoId = res.videoId;
				}
				uploadProgress = Math.round(((chunkIndex + 1) / totalChunks) * 100);
			}

			statusMessage = 'Creating record...';
			const finalData = new FormData();
			finalData.append('videoId', finalVideoId);
			finalData.append('title', title);
			finalData.append('description', description);
			finalData.append('category', category);
			finalData.append('visibility', 'public');
			finalData.append('isShort', isShort.toString());

			await fetchApi('/videos/create-record', {
				method: 'POST',
				body: finalData
			});

			fileId = serverFileId;
			success = true;
		} catch (err: any) {
			error = err.message || 'Upload failed.';
		} finally {
			isUploading = false;
		}
	}

	async function uploadRemote(e: Event) {
		e.preventDefault();
		if (!videoUrl) {
			error = 'Please enter a video URL.';
			return;
		}

		isUploading = true;
		error = '';
		uploadProgress = 0;
		statusMessage = 'Queuing remote upload...';

		try {
			const startRes = await fetchApi('/videos/remote-upload/start', {
				method: 'POST',
				body: JSON.stringify({ videoUrl })
			});
			const remoteId = startRes.id;
			if (!remoteId) throw new Error("Failed to get remote upload ID.");

			statusMessage = 'Waiting in queue...';

			// Polling
			while (true) {
				await new Promise(r => setTimeout(r, 4000));
				const statusRes = await fetchApi(`/videos/remote-upload/status?id=${remoteId}`);
				const statusData = statusRes[remoteId];

				if (statusData) {
					if (statusData.status === 'finished') {
						uploadProgress = 100;
						statusMessage = 'Publishing video...';
						const finalVideoId = statusData.linkid;
						
						const finalData = new FormData();
						finalData.append('videoId', finalVideoId);
						finalData.append('title', title);
						finalData.append('description', description);
						finalData.append('category', category);
						finalData.append('visibility', 'public');
						finalData.append('isShort', isShort.toString());

						await fetchApi('/videos/create-record', {
							method: 'POST',
							body: finalData
						});
						
						// In remote uploads, linkid is the fileId used for URLs
						fileId = finalVideoId;
						success = true;
						break;
					} else if (statusData.status === 'error') {
						throw new Error(statusData.error_message || 'Remote download error.');
					} else {
						const loaded = statusData.bytes_loaded || 0;
						const total = statusData.bytes_total || 0;
						if (total > 0) {
							uploadProgress = Math.floor((loaded / total) * 100);
							statusMessage = `Downloading... ${uploadProgress}%`;
						}
					}
				}
			}

		} catch (err: any) {
			error = err.message || 'Remote upload failed.';
		} finally {
			isUploading = false;
		}
	}

	function reset() {
		file = null;
		title = '';
		description = '';
		category = 'Other';
		isShort = false;
		videoUrl = '';
		isUploading = false;
		uploadProgress = 0;
		error = '';
		success = false;
		fileId = '';
		statusMessage = '';
	}
</script>

<svelte:head>
	<title>Upload Video - Aurahub</title>
</svelte:head>

<div class="max-w-4xl mx-auto py-8 px-4 sm:px-6">
	<div class="bg-card border rounded-2xl shadow-sm overflow-hidden">
		<div class="p-6 border-b bg-muted/20 text-center">
			<h1 class="text-2xl font-bold tracking-tight">Upload Video</h1>
			<p class="text-muted-foreground text-sm mt-1">Share your content with the Aurahub community.</p>
		</div>

		<div class="p-4 sm:p-6 flex justify-center border-b bg-muted/5">
			<div class="flex bg-muted p-1 rounded-xl w-full max-w-2xl">
				<button 
					onclick={() => uploadType = 'direct'} 
					class={`flex-1 py-2 text-sm font-semibold rounded-lg flex items-center justify-center gap-2 transition-all ${uploadType === 'direct' ? 'bg-background shadow text-foreground' : 'text-muted-foreground hover:text-foreground'}`}
				>
					<UploadCloud class="w-4 h-4" /> Direct Upload
				</button>
				<button 
					onclick={() => uploadType = 'remote'} 
					class={`flex-1 py-2 text-sm font-semibold rounded-lg flex items-center justify-center gap-2 transition-all ${uploadType === 'remote' ? 'bg-background shadow text-foreground' : 'text-muted-foreground hover:text-foreground'}`}
				>
					<LinkIcon class="w-4 h-4" /> Remote URL
				</button>
				<button 
					onclick={() => uploadType = 'playlist'} 
					class={`flex-1 py-2 text-sm font-semibold rounded-lg flex items-center justify-center gap-2 transition-all ${uploadType === 'playlist' ? 'bg-background shadow text-foreground' : 'text-muted-foreground hover:text-foreground'}`}
				>
					<ListVideo class="w-4 h-4" /> Playlist Batch
				</button>
			</div>
		</div>

		<div class="p-6 sm:p-8">
			{#if success}
				<div class="flex flex-col items-center justify-center py-12 text-center animate-in zoom-in duration-300">
					<div class="w-20 h-20 bg-green-100 text-green-600 rounded-full flex items-center justify-center mb-6">
						<CheckCircle class="w-10 h-10" />
					</div>
					<h2 class="text-2xl font-bold mb-2">Upload Complete!</h2>
					<p class="text-muted-foreground max-w-md mb-8">
						Your video has been successfully uploaded and is now processing.
					</p>
					<div class="flex gap-4">
						<button onclick={reset} class="px-6 py-2 rounded-full border hover:bg-muted font-medium transition-colors">
							Upload Another
						</button>
						<a href={`/watch/${fileId}`} class="px-6 py-2 rounded-full bg-primary text-primary-foreground hover:bg-primary/90 font-medium transition-colors">
							View Video
						</a>
					</div>
				</div>
			{:else if uploadType === 'playlist'}
				<PlaylistUploadForm {playlists} />
			{:else}
				<form onsubmit={uploadType === 'direct' ? uploadDirect : uploadRemote} class="space-y-8 max-w-2xl mx-auto">
					{#if error}
						<div class="p-4 bg-red-50 text-red-600 rounded-lg text-sm font-medium flex items-center justify-between">
							{error}
							<button type="button" onclick={() => error = ''}><X class="w-4 h-4" /></button>
						</div>
					{/if}

					<!-- Common Fields -->
					<div class="space-y-4">
						<div class="space-y-1.5">
							<label for="title" class="text-sm font-medium">Title <span class="text-red-500">*</span></label>
							<input type="text" id="title" bind:value={title} required disabled={isUploading} class="w-full h-10 px-3 rounded-md border border-input bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary" placeholder="Video Title" />
						</div>

						{#if uploadType === 'remote'}
							<div class="space-y-1.5">
								<label for="url" class="text-sm font-medium">Remote URL (Direct MP4 / Stream URL) <span class="text-red-500">*</span></label>
								<input type="url" id="url" bind:value={videoUrl} required disabled={isUploading} class="w-full h-10 px-3 rounded-md border border-input bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary font-mono" placeholder="https://example.com/video.mp4" />
							</div>
						{:else}
							<div class="space-y-1.5">
								<label class="text-sm font-medium">Video File <span class="text-red-500">*</span></label>
								<div class="relative">
									<input type="file" id="videoFile" accept="video/mp4,video/webm,video/ogg" class="hidden" onchange={handleFileSelect} disabled={isUploading} />
									<label for="videoFile" class="flex flex-col items-center justify-center w-full h-40 border-2 border-dashed rounded-xl cursor-pointer hover:bg-muted/50 transition-colors border-muted-foreground/30 bg-background/50">
										<div class="flex flex-col items-center justify-center pt-5 pb-6 text-center px-4">
											{#if file}
												<FileVideo class="w-10 h-10 text-primary mb-3" />
												<p class="text-sm font-semibold">{file.name}</p>
												<p class="text-xs text-muted-foreground mt-1">{(file.size / (1024 * 1024)).toFixed(2)} MB</p>
											{:else}
												<UploadCloud class="w-10 h-10 text-muted-foreground mb-3 opacity-50" />
												<p class="text-sm font-semibold mb-1">Click to select video</p>
												<p class="text-xs text-muted-foreground">MP4, WEBM, or OGG</p>
											{/if}
										</div>
									</label>
								</div>
							</div>
						{/if}

						<div class="grid grid-cols-2 gap-4">
							<div class="space-y-1.5">
								<label for="category" class="text-sm font-medium">Category</label>
								<select id="category" bind:value={category} disabled={isUploading} class="w-full h-10 px-3 rounded-md border border-input bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary">
									<option value="Other">Other</option>
									<option value="Gaming">Gaming</option>
									<option value="Education">Education</option>
									<option value="Entertainment">Entertainment</option>
									<option value="Music">Music</option>
									<option value="Science">Science & Tech</option>
								</select>
							</div>
						</div>

						<div class="space-y-1.5">
							<label for="description" class="text-sm font-medium">Description</label>
							<textarea id="description" bind:value={description} disabled={isUploading} rows="4" class="w-full p-3 rounded-md border border-input bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary resize-y" placeholder="Tell viewers about your video"></textarea>
						</div>

						<div class="flex items-center space-x-2">
							<input type="checkbox" id="isShort" bind:checked={isShort} disabled={isUploading} class="w-4 h-4 text-primary rounded border-input focus:ring-primary" />
							<label for="isShort" class="text-sm font-medium cursor-pointer">Upload as Short (Vertical Video)</label>
						</div>
					</div>

					{#if isUploading}
						<div class="space-y-2 p-4 bg-muted/30 rounded-xl border">
							<div class="flex justify-between text-sm font-medium">
								<span>{statusMessage}</span>
								<span>{uploadProgress}%</span>
							</div>
							<div class="w-full bg-secondary rounded-full h-2 overflow-hidden">
								<div class="bg-primary h-2 transition-all duration-300" style="width: {uploadProgress}%"></div>
							</div>
						</div>
					{/if}

					<div class="pt-4">
						<button type="submit" disabled={isUploading || (uploadType === 'direct' && !file)} class="w-full h-12 flex items-center justify-center bg-primary text-primary-foreground font-semibold rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
							{#if isUploading}
								<div class="w-5 h-5 border-2 border-white/30 border-t-white rounded-full animate-spin mr-2"></div>
								Processing...
							{:else}
								Upload Video
							{/if}
						</button>
					</div>
				</form>
			{/if}
		</div>
	</div>
</div>
