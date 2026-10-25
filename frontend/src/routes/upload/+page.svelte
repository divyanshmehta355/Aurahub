<script lang="ts">
	import { onDestroy } from 'svelte';
	import { fetchApi } from '#lib/api';
	import {
		UploadCloud,
		FileVideo,
		X,
		CheckCircle,
		CheckCircle2,
		AlertCircle,
		Link as LinkIcon,
		ListVideo,
		Loader2,
		Hash,
		Plus
	} from 'lucide-svelte';
	import PlaylistUploadForm from '#lib/components/PlaylistUploadForm.svelte';
	import { Button } from '#lib/components/ui/button';
	import { Input } from '#lib/components/ui/input';
	import { Label } from '#lib/components/ui/label';
	import { Textarea } from '#lib/components/ui/textarea';
	import * as Card from '#lib/components/ui/card';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';

	const queryClient = useQueryClient();

	let uploadType = $state('direct'); // 'direct', 'remote', 'playlist'

	// Fetch playlists for PlaylistBatch mode if needed
	const playlistsQuery = createQuery(
		() => ({
			queryKey: ['my-playlists'],
			queryFn: async () => {
				const res = await fetchApi('/playlists/my-playlists');
				return res || [];
			}
		}),
		() => queryClient
	);

	// Common form
	let title = $state('');
	let description = $state('');
	let category = $state('Other');
	let isShort = $state(false);
	let isAdult = $state(false);

	// Direct
	let file: File | null = $state(null);
	let fileError = $state('');
	let isDragging = $state(false);
	let dragOverValid = $state(true);
	let dragCounter = 0;

	// Remote
	let videoUrl = $state('');

	let isUploading = $state(false);
	let uploadProgress = $state(0);
	let uploadedBytes = $state(0);
	let totalBytes = $state(0);
	let uploadSpeed = $state(0); // bytes/sec
	let etaSeconds = $state<number | null>(null);
	let error = $state('');
	let success = $state(false);
	let fileId = $state('');
	let statusMessage = $state('');

	// Upload Cancellation State
	let currentXhr = $state<XMLHttpRequest | null>(null);
	let isCancelled = $state(false);

	function cancelUpload() {
		isCancelled = true;
		if (currentXhr) {
			try {
				currentXhr.abort();
			} catch {}
			currentXhr = null;
		}
		isUploading = false;
		uploadProgress = 0;
		uploadedBytes = 0;
		uploadSpeed = 0;
		etaSeconds = null;
		statusMessage = '';
		error = 'Upload was cancelled.';
	}

	onDestroy(() => {
		if (currentXhr) {
			try {
				currentXhr.abort();
			} catch {}
			currentXhr = null;
		}
	});

	function formatBytes(bytes: number): string {
		if (bytes <= 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
	}

	function formatSpeed(bytesPerSec: number): string {
		if (bytesPerSec <= 0) return '0 KB/s';
		return `${formatBytes(bytesPerSec)}/s`;
	}

	function formatEta(seconds: number | null): string {
		if (seconds === null || !isFinite(seconds) || seconds <= 0) return '--';
		if (seconds < 60) return `${Math.ceil(seconds)}s left`;
		const mins = Math.floor(seconds / 60);
		const secs = Math.ceil(seconds % 60);
		return `${mins}m ${secs}s left`;
	}

	// Tags State
	let tags = $state<string[]>([]);
	let tagInput = $state('');

	const suggestedTags = [
		'gaming',
		'tutorial',
		'music',
		'tech',
		'vlog',
		'entertainment',
		'coding',
		'review',
		'comedy',
		'education'
	];

	function addTag(rawTag: string) {
		const clean = rawTag.trim().replace(/^#+/, '').toLowerCase();
		if (!clean) return;
		if (tags.length >= 15) {
			error = 'Maximum 15 tags allowed per video.';
			return;
		}
		if (!tags.includes(clean)) {
			tags = [...tags, clean];
			error = '';
		}
		tagInput = '';
	}

	function removeTag(tagToRemove: string) {
		tags = tags.filter((t) => t !== tagToRemove);
	}

	function handleTagKeyDown(e: KeyboardEvent) {
		if (e.key === 'Enter' || e.key === ',') {
			e.preventDefault();
			addTag(tagInput);
		} else if (e.key === 'Backspace' && tagInput === '' && tags.length > 0) {
			removeTag(tags[tags.length - 1]);
		}
	}

	const validExtensions = ['mp4', 'webm', 'ogg', 'mov', 'mkv', 'avi', 'm4v'];

	function validateAndProcessFile(selectedFile: File): boolean {
		fileError = '';
		const extension = selectedFile.name.split('.').pop()?.toLowerCase() || '';
		const isVideoMime = selectedFile.type.startsWith('video/');
		const isValid = isVideoMime || validExtensions.includes(extension);

		if (!isValid) {
			fileError = `Unsupported file format: "${selectedFile.name}". Please upload a supported video file (${validExtensions.slice(0, 5).map((e) => e.toUpperCase()).join(', ')}).`;
			error = fileError;
			return false;
		}

		file = selectedFile;
		fileError = '';
		error = '';
		if (!title.trim()) {
			title = selectedFile.name.replace(/\.[^/.]+$/, '');
		}
		return true;
	}

	function clearSelectedFile(e?: Event) {
		if (e) {
			e.preventDefault();
			e.stopPropagation();
		}
		file = null;
		fileError = '';
		const input = document.getElementById('videoFile') as HTMLInputElement | null;
		if (input) input.value = '';
	}

	function handleFileSelect(e: Event) {
		const target = e.target as HTMLInputElement;
		if (target.files && target.files.length > 0) {
			validateAndProcessFile(target.files[0]);
		}
	}

	function handleDragEnter(e: DragEvent) {
		e.preventDefault();
		if (isUploading) return;
		dragCounter++;
		isDragging = true;

		if (e.dataTransfer && e.dataTransfer.items) {
			const items = Array.from(e.dataTransfer.items);
			const hasInvalid = items.some(
				(item) => item.kind === 'file' && item.type && !item.type.startsWith('video/')
			);
			dragOverValid = !hasInvalid;
		} else {
			dragOverValid = true;
		}
	}

	function handleDragOver(e: DragEvent) {
		e.preventDefault();
		if (isUploading) return;
		if (e.dataTransfer) {
			e.dataTransfer.dropEffect = dragOverValid ? 'copy' : 'none';
		}
	}

	function handleDragLeave(e: DragEvent) {
		e.preventDefault();
		dragCounter--;
		if (dragCounter <= 0) {
			dragCounter = 0;
			isDragging = false;
			dragOverValid = true;
		}
	}

	function handleDrop(e: DragEvent) {
		e.preventDefault();
		dragCounter = 0;
		isDragging = false;
		dragOverValid = true;

		if (isUploading) return;

		if (e.dataTransfer && e.dataTransfer.files && e.dataTransfer.files.length > 0) {
			validateAndProcessFile(e.dataTransfer.files[0]);
		}
	}

	async function uploadDirect(e: Event) {
		e.preventDefault();
		if (!file) {
			error = 'Please select a video file.';
			return;
		}

		isUploading = true;
		isCancelled = false;
		error = '';
		uploadProgress = 0;
		uploadedBytes = 0;
		totalBytes = file.size;
		uploadSpeed = 0;
		etaSeconds = null;
		statusMessage = 'Requesting direct upload URL...';

		try {
			// 1. Get the direct upload URL from the Aura API
			const initRes = await fetchApi('/videos/get-upload-url');
			if (isCancelled) return;

			const targetUploadUrl = initRes?.url || initRes?.result?.url || initRes?.uploadUrl;
			if (!targetUploadUrl) {
				throw new Error('Failed to obtain direct upload URL from service.');
			}

			// 2. Directly upload the video file to the upload URL with real-time progress & speed
			statusMessage = 'Uploading video...';
			const uploadResult = await new Promise<any>((resolve, reject) => {
				const xhr = new XMLHttpRequest();
				currentXhr = xhr;
				xhr.open('POST', targetUploadUrl);

				let lastLoaded = 0;
				let lastTime = performance.now();
				const speedSamples: number[] = [];

				xhr.upload.onloadstart = (event) => {
					lastTime = performance.now();
					lastLoaded = 0;
					uploadedBytes = 0;
					totalBytes = event.total || file!.size;
				};

				xhr.upload.onprogress = (event) => {
					if (isCancelled) return;
					if (event.lengthComputable) {
						const now = performance.now();
						uploadedBytes = event.loaded;
						totalBytes = event.total;
						uploadProgress = Math.round((event.loaded / event.total) * 100);

						const timeDiff = (now - lastTime) / 1000;
						if (timeDiff >= 0.2) {
							const bytesDiff = event.loaded - lastLoaded;
							const currentSpeed = bytesDiff / timeDiff;

							speedSamples.push(currentSpeed);
							if (speedSamples.length > 5) speedSamples.shift();
							const avgSpeed = speedSamples.reduce((a, b) => a + b, 0) / speedSamples.length;

							uploadSpeed = avgSpeed;
							const remainingBytes = Math.max(0, event.total - event.loaded);
							etaSeconds = avgSpeed > 0 ? remainingBytes / avgSpeed : null;

							lastLoaded = event.loaded;
							lastTime = now;
						}
					}
				};

				xhr.onload = () => {
					currentXhr = null;
					if (isCancelled) {
						reject(new Error('Upload cancelled.'));
						return;
					}
					if (xhr.status >= 200 && xhr.status < 300) {
						uploadProgress = 100;
						uploadedBytes = totalBytes;
						uploadSpeed = 0;
						etaSeconds = 0;
						try {
							const json = JSON.parse(xhr.responseText);
							resolve(json);
						} catch {
							resolve(xhr.responseText);
						}
					} else {
						reject(new Error(`Upload service returned HTTP ${xhr.status}: ${xhr.statusText || 'Error'}`));
					}
				};

				xhr.onerror = () => {
					currentXhr = null;
					if (isCancelled) {
						reject(new Error('Upload cancelled.'));
					} else {
						reject(new Error('Network error occurred while uploading video.'));
					}
				};

				xhr.onabort = () => {
					currentXhr = null;
					reject(new Error('Upload cancelled.'));
				};

				xhr.ontimeout = () => {
					currentXhr = null;
					reject(new Error('Upload timed out. Please try again.'));
				};

				const formData = new FormData();
				formData.append('file', file!);
				xhr.send(formData);
			});

			if (isCancelled) return;

			// Extract file ID returned by the upload service
			const finalVideoId =
				uploadResult?.result?.id ||
				uploadResult?.id ||
				uploadResult?.videoId ||
				(typeof uploadResult === 'string' && uploadResult.trim());

			if (!finalVideoId) {
				throw new Error('Upload completed, but no video ID was received from the service.');
			}

			// 3. Create the record in Aurahub database
			statusMessage = 'Finalizing video publishing...';
			const finalData = new FormData();
			finalData.append('videoId', finalVideoId);
			finalData.append('title', title);
			finalData.append('description', description);
			finalData.append('category', category);
			finalData.append('visibility', 'public');
			finalData.append('isShort', isShort.toString());
			finalData.append('isAdult', isAdult.toString());
			if (tags.length > 0) {
				finalData.append('tags', JSON.stringify(tags));
			}

			await fetchApi('/videos/create-record', {
				method: 'POST',
				body: finalData
			});

			fileId = finalVideoId;
			success = true;
		} catch (err: any) {
			if (isCancelled || err.message === 'Upload cancelled.') {
				error = 'Upload was cancelled.';
			} else {
				error = err.message || 'Upload failed.';
			}
		} finally {
			currentXhr = null;
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
		isCancelled = false;
		error = '';
		uploadProgress = 0;
		uploadedBytes = 0;
		totalBytes = 0;
		uploadSpeed = 0;
		etaSeconds = null;
		statusMessage = 'Queuing remote upload...';

		try {
			const startRes = await fetchApi('/videos/remote-upload/start', {
				method: 'POST',
				body: JSON.stringify({ videoUrl })
			});
			if (isCancelled) return;
			const remoteId = startRes.id;
			if (!remoteId) throw new Error('Failed to get remote upload ID.');

			statusMessage = 'Waiting in queue...';

			// Polling
			while (!isCancelled && isUploading) {
				await new Promise((r) => setTimeout(r, 4000));
				if (isCancelled || !isUploading) break;
				const statusRes = await fetchApi(`/videos/remote-upload/status?id=${remoteId}`);
				if (isCancelled || !isUploading) break;
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
						finalData.append('isAdult', isAdult.toString());
						if (tags.length > 0) {
							finalData.append('tags', JSON.stringify(tags));
						}

						await fetchApi('/videos/create-record', {
							method: 'POST',
							body: finalData
						});

						fileId = finalVideoId;
						success = true;
						break;
					} else if (statusData.status === 'error') {
						throw new Error(statusData.error_message || 'Remote download error.');
					} else {
						const loaded = statusData.bytes_loaded || 0;
						const total = statusData.bytes_total || 0;
						if (total > 0) {
							uploadedBytes = loaded;
							totalBytes = total;
							uploadProgress = Math.floor((loaded / total) * 100);
							statusMessage = `Remote transfer in progress...`;
						}
					}
				}
			}
		} catch (err: any) {
			if (isCancelled || err.message === 'Upload cancelled.') {
				error = 'Upload was cancelled.';
			} else {
				error = err.message || 'Remote upload failed.';
			}
		} finally {
			isUploading = false;
		}
	}

	function reset() {
		if (currentXhr) {
			try {
				currentXhr.abort();
			} catch {}
			currentXhr = null;
		}
		isCancelled = false;
		file = null;
		fileError = '';
		isDragging = false;
		dragOverValid = true;
		dragCounter = 0;
		title = '';
		description = '';
		category = 'Other';
		isShort = false;
		isAdult = false;
		tags = [];
		tagInput = '';
		videoUrl = '';
		isUploading = false;
		uploadProgress = 0;
		uploadedBytes = 0;
		totalBytes = 0;
		uploadSpeed = 0;
		etaSeconds = null;
		error = '';
		success = false;
		fileId = '';
		statusMessage = '';
	}
</script>

<svelte:head>
	<title>Upload Video - Aurahub</title>
</svelte:head>

<div class="mx-auto max-w-4xl px-4 py-8 sm:px-6">
	<Card.Root class="overflow-hidden shadow-sm">
		<div class="bg-muted/20 border-b p-6 text-center">
			<h1 class="text-2xl font-bold tracking-tight">Upload Video</h1>
			<p class="text-muted-foreground mt-1 text-sm">
				Share your content with the Aurahub community.
			</p>
		</div>

		<div class="bg-muted/5 flex justify-center border-b p-4 sm:p-6">
			<div class="bg-muted flex w-full max-w-2xl rounded-xl p-1">
				<button
					onclick={() => (uploadType = 'direct')}
					class={`flex flex-1 items-center justify-center gap-2 rounded-lg py-2 text-sm font-semibold transition-all ${uploadType === 'direct' ? 'text-foreground bg-background shadow' : 'text-muted-foreground hover:text-foreground'}`}
				>
					<UploadCloud class="h-4 w-4" /> Direct Upload
				</button>
				<button
					onclick={() => (uploadType = 'remote')}
					class={`flex flex-1 items-center justify-center gap-2 rounded-lg py-2 text-sm font-semibold transition-all ${uploadType === 'remote' ? 'text-foreground bg-background shadow' : 'text-muted-foreground hover:text-foreground'}`}
				>
					<LinkIcon class="h-4 w-4" /> Remote URL
				</button>
				<button
					onclick={() => (uploadType = 'playlist')}
					class={`flex flex-1 items-center justify-center gap-2 rounded-lg py-2 text-sm font-semibold transition-all ${uploadType === 'playlist' ? 'text-foreground bg-background shadow' : 'text-muted-foreground hover:text-foreground'}`}
				>
					<ListVideo class="h-4 w-4" /> Bulk / Playlist Batch
				</button>
			</div>
		</div>

		<Card.Content class="p-6 sm:p-8">
			{#if success}
				<div
					class="animate-in zoom-in flex flex-col items-center justify-center py-12 text-center duration-300"
				>
					<div
						class="mb-6 flex h-20 w-20 items-center justify-center rounded-full bg-green-100 text-green-600"
					>
						<CheckCircle class="h-10 w-10" />
					</div>
					<h2 class="mb-2 text-2xl font-bold">Upload Complete!</h2>
					<p class="text-muted-foreground mb-8 max-w-md">
						Your video has been successfully uploaded and is now processing.
					</p>
					<div class="flex gap-4">
						<Button variant="outline" class="rounded-full px-6" onclick={reset}>
							Upload Another
						</Button>
						<a href={`/watch/${fileId}`}>
							<Button class="rounded-full px-6">View Video</Button>
						</a>
					</div>
				</div>
			{:else if uploadType === 'playlist'}
				<PlaylistUploadForm playlists={playlistsQuery.data || []} />
			{:else}
				<form
					onsubmit={uploadType === 'direct' ? uploadDirect : uploadRemote}
					class="mx-auto max-w-2xl space-y-8"
				>
					{#if error}
						<div
							class="bg-destructive/10 text-destructive flex items-center justify-between rounded-lg p-4 text-sm font-medium"
						>
							{error}
							<button type="button" onclick={() => (error = '')}><X class="h-4 w-4" /></button>
						</div>
					{/if}

					<div class="space-y-4">
						<div class="space-y-1.5">
							<Label for="title">Title <span class="text-destructive">*</span></Label>
							<Input
								type="text"
								id="title"
								bind:value={title}
								required
								disabled={isUploading}
								placeholder="Video Title"
							/>
						</div>

						{#if uploadType === 'remote'}
							<div class="space-y-1.5">
								<Label for="url"
									>Remote URL (Direct MP4 / Stream URL) <span class="text-destructive">*</span
									></Label
								>
								<Input
									type="url"
									id="url"
									bind:value={videoUrl}
									required
									disabled={isUploading}
									placeholder="https://example.com/video.mp4"
									class="font-mono"
								/>
							</div>
						{:else}
							<div class="space-y-2">
								<div class="flex items-center justify-between">
									<Label for="videoFile" class="text-sm font-medium">
										Video File <span class="text-destructive">*</span>
									</Label>
									{#if file}
										<span class="text-xs text-muted-foreground font-mono">
											{(file.size / (1024 * 1024)).toFixed(2)} MB
										</span>
									{/if}
								</div>

								<div
									role="region"
									aria-label="Video file drop zone"
									ondragenter={handleDragEnter}
									ondragover={handleDragOver}
									ondragleave={handleDragLeave}
									ondrop={handleDrop}
									class="relative w-full transition-all duration-200"
								>
									<input
										type="file"
										id="videoFile"
										accept="video/*,video/mp4,video/webm,video/ogg,video/quicktime,video/x-matroska"
										class="hidden"
										onchange={handleFileSelect}
										disabled={isUploading}
									/>

									{#if isDragging}
										<!-- Active Drag-Over Overlay State -->
										<div
											class={`flex min-h-[170px] w-full flex-col items-center justify-center rounded-2xl border-2 border-dashed p-6 text-center transition-all duration-200 shadow-lg ${
												dragOverValid
													? 'border-primary bg-primary/10 ring-4 ring-primary/20 scale-[1.01]'
													: 'border-destructive bg-destructive/10 ring-4 ring-destructive/20 animate-pulse'
											}`}
										>
											{#if dragOverValid}
												<div class="mb-3 flex h-14 w-14 items-center justify-center rounded-2xl bg-primary/20 text-primary shadow-sm">
													<UploadCloud class="h-8 w-8 animate-bounce text-primary" />
												</div>
												<p class="text-base font-bold text-foreground">Drop video here to upload</p>
												<span class="mt-1.5 inline-flex items-center gap-1.5 rounded-full bg-primary/20 px-3 py-0.5 text-xs font-semibold text-primary">
													<CheckCircle2 class="h-3.5 w-3.5" />
													Supported video format
												</span>
											{:else}
												<div class="mb-3 flex h-14 w-14 items-center justify-center rounded-2xl bg-destructive/20 text-destructive shadow-sm">
													<AlertCircle class="h-8 w-8 animate-pulse text-destructive" />
												</div>
												<p class="text-base font-bold text-destructive">Unsupported file type</p>
												<span class="mt-1.5 inline-flex items-center gap-1.5 rounded-full bg-destructive/20 px-3 py-0.5 text-xs font-semibold text-destructive">
													Only video files (MP4, WebM, OGG, MOV, MKV) are allowed
												</span>
											{/if}
										</div>
									{:else if file}
										<!-- Selected File Showcase Card -->
										<div class="relative flex w-full flex-col sm:flex-row items-start sm:items-center justify-between gap-4 rounded-2xl border-2 border-primary/40 bg-gradient-to-r from-primary/5 via-background to-primary/5 p-4 sm:p-5 shadow-sm transition-all">
											<div class="flex items-center gap-3.5 min-w-0 flex-1">
												<div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-primary/15 text-primary border border-primary/20 shadow-inner">
													<FileVideo class="h-6 w-6" />
												</div>
												<div class="min-w-0 flex-1">
													<div class="flex items-center gap-2">
														<p class="text-sm font-bold text-foreground truncate max-w-[240px] sm:max-w-md" title={file.name}>
															{file.name}
														</p>
														<span class="shrink-0 rounded-md border border-primary/30 bg-primary/10 px-1.5 py-0.5 font-mono text-[10px] font-bold text-primary uppercase">
															{file.name.split('.').pop() || 'VIDEO'}
														</span>
													</div>
													<div class="mt-1 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
														<span class="font-mono font-medium text-foreground">
															{(file.size / (1024 * 1024)).toFixed(2)} MB
														</span>
														<span>•</span>
														<span class="inline-flex items-center gap-1 font-semibold text-emerald-600 dark:text-emerald-400">
															<CheckCircle2 class="h-3 w-3" /> Ready to upload
														</span>
													</div>
												</div>
											</div>

											<div class="flex items-center gap-2 self-end sm:self-center shrink-0">
												<label
													for="videoFile"
													class="inline-flex h-8 cursor-pointer items-center justify-center rounded-lg border border-border bg-background px-3 text-xs font-semibold text-foreground transition-colors hover:bg-muted hover:text-foreground"
												>
													Change
												</label>
												<Button
													type="button"
													variant="ghost"
													size="sm"
													disabled={isUploading}
													class="h-8 w-8 p-0 text-muted-foreground hover:bg-destructive/10 hover:text-destructive transition-colors"
													onclick={clearSelectedFile}
													title="Remove selected file"
												>
													<X class="h-4 w-4" />
												</Button>
											</div>
										</div>
									{:else}
										<!-- Empty Idle Drop Zone -->
										<label
											for="videoFile"
											class={`group relative flex min-h-[160px] w-full cursor-pointer flex-col items-center justify-center rounded-2xl border-2 border-dashed bg-background/50 p-6 text-center transition-all duration-200 hover:border-primary/60 hover:bg-muted/30 ${
												fileError ? 'border-destructive/60 bg-destructive/5' : 'border-border/80'
											}`}
										>
											<div class="flex flex-col items-center justify-center">
												<div class="mb-3 flex h-12 w-12 items-center justify-center rounded-2xl bg-muted text-muted-foreground transition-all duration-200 group-hover:scale-110 group-hover:bg-primary/10 group-hover:text-primary">
													<UploadCloud class="h-6 w-6 transition-transform group-hover:-translate-y-0.5" />
												</div>
												<p class="text-sm font-semibold text-foreground">
													Drag & drop video here, or <span class="text-primary underline decoration-primary/40 underline-offset-2 group-hover:decoration-primary">browse files</span>
												</p>
												<p class="mt-1 text-xs text-muted-foreground">
													Direct high-speed video upload
												</p>

												<!-- Supported Format Pill Chips -->
												<div class="mt-3 flex flex-wrap items-center justify-center gap-1.5">
													{#each ['MP4', 'WebM', 'OGG', 'MOV', 'MKV'] as fmt}
														<span class="rounded-md border border-border/70 bg-muted/60 px-2 py-0.5 font-mono text-[10px] font-semibold text-muted-foreground transition-colors group-hover:border-primary/30 group-hover:text-foreground">
															{fmt}
														</span>
													{/each}
												</div>
											</div>
										</label>
									{/if}

									<!-- Inline Format Validation Error Feedback -->
									{#if fileError}
										<div class="mt-2 flex items-center justify-between rounded-xl border border-destructive/30 bg-destructive/10 px-3.5 py-2 text-xs font-medium text-destructive animate-in fade-in slide-in-from-top-1 duration-150">
											<div class="flex items-center gap-2">
												<AlertCircle class="h-4 w-4 shrink-0" />
												<span>{fileError}</span>
											</div>
											<button
												type="button"
												onclick={() => (fileError = '')}
												class="hover:opacity-80 p-0.5"
												title="Dismiss error"
											>
												<X class="h-3.5 w-3.5" />
											</button>
										</div>
									{/if}
								</div>
							</div>
						{/if}

						<div class="grid grid-cols-2 gap-4">
							<div class="space-y-1.5">
								<Label for="category">Category</Label>
								<select
									id="category"
									bind:value={category}
									disabled={isUploading}
									class="border-input h-10 w-full rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-primary focus:outline-none disabled:opacity-50"
								>
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
							<Label for="description">Description</Label>
							<Textarea
								id="description"
								bind:value={description}
								disabled={isUploading}
								rows={4}
								placeholder="Tell viewers about your video"
							/>
						</div>

						<!-- Interactive Tag Chips -->
						<div class="space-y-2">
							<div class="flex items-center justify-between">
								<Label class="flex items-center gap-1.5 text-sm font-medium">
									<Hash class="h-3.5 w-3.5 text-primary" />
									Tags
								</Label>
								<span class="text-xs text-muted-foreground">{tags.length}/15 tags</span>
							</div>

							<!-- Tag Input Box with Chips -->
							<div
								class="border-input focus-within:ring-2 focus-within:ring-primary focus-within:border-primary flex min-h-[46px] w-full flex-wrap items-center gap-2 rounded-xl border bg-background/50 p-2 text-sm transition-all"
							>
								{#each tags as tag (tag)}
									<span
										class="inline-flex items-center gap-1 rounded-lg border border-primary/30 bg-primary/10 px-2.5 py-1 text-xs font-semibold text-primary transition-all animate-in zoom-in-95 duration-150"
									>
										<span>#{tag}</span>
										<button
											type="button"
											onclick={() => removeTag(tag)}
											class="text-primary/70 hover:text-primary hover:bg-primary/20 -mr-0.5 ml-0.5 rounded-full p-0.5 transition-colors"
											title={`Remove #${tag}`}
											aria-label={`Remove #${tag}`}
										>
											<X class="h-3 w-3" />
										</button>
									</span>
								{/each}

								<input
									type="text"
									bind:value={tagInput}
									onkeydown={handleTagKeyDown}
									placeholder={tags.length === 0
										? 'Type tag and press Enter or comma...'
										: 'Add tag...'}
									disabled={isUploading || tags.length >= 15}
									class="min-w-[130px] flex-1 bg-transparent px-2 py-1 text-xs text-foreground placeholder:text-muted-foreground/60 focus:outline-none disabled:opacity-50"
								/>

								{#if tagInput.trim().length > 0}
									<Button
										type="button"
										variant="ghost"
										size="sm"
										class="h-7 gap-1 rounded-lg px-2 text-xs font-semibold text-primary hover:bg-primary/10"
										onclick={() => addTag(tagInput)}
									>
										<Plus class="h-3 w-3" /> Add
									</Button>
								{/if}
							</div>

							<!-- Suggested Tag Quick-Pills -->
							<div class="flex flex-wrap items-center gap-1.5 pt-0.5">
								<span class="text-[11px] font-medium text-muted-foreground">Suggested:</span>
								{#each suggestedTags as suggestion}
									{#if !tags.includes(suggestion)}
										<button
											type="button"
											onclick={() => addTag(suggestion)}
											disabled={isUploading || tags.length >= 15}
											class="rounded-md border border-border/70 bg-muted/40 px-2 py-0.5 text-[11px] font-medium text-muted-foreground transition-colors hover:border-primary/50 hover:bg-primary/10 hover:text-primary disabled:opacity-40"
										>
											+{suggestion}
										</button>
									{/if}
								{/each}
							</div>
						</div>

						<div class="flex flex-col gap-3">
							<div class="flex items-center space-x-2">
								<input
									type="checkbox"
									id="isShort"
									bind:checked={isShort}
									disabled={isUploading}
									class="border-input h-4 w-4 rounded text-primary focus:ring-primary disabled:opacity-50"
								/>
								<Label for="isShort" class="cursor-pointer">Upload as Short (Vertical Video)</Label>
							</div>
							<div class="flex items-center space-x-2">
								<input
									type="checkbox"
									id="isAdult"
									bind:checked={isAdult}
									disabled={isUploading}
									class="border-input h-4 w-4 rounded text-primary focus:ring-primary disabled:opacity-50"
								/>
								<Label for="isAdult" class="cursor-pointer">Contains Adult Content</Label>
							</div>
						</div>
					</div>

					{#if isUploading}
						<div class="rounded-2xl border border-primary/25 bg-muted/40 p-5 shadow-sm space-y-4">
							<div class="flex items-center justify-between">
								<div class="flex items-center gap-3">
									<div class="flex h-10 w-10 items-center justify-center rounded-xl bg-primary/10 text-primary">
										<UploadCloud class="h-5 w-5 animate-bounce" />
									</div>
									<div>
										<p class="text-sm font-bold text-foreground">{statusMessage}</p>
										<p class="text-xs text-muted-foreground truncate max-w-[220px] sm:max-w-md">
											{file?.name || videoUrl}
										</p>
									</div>
								</div>
								<div class="flex items-center gap-3">
									<span class="font-mono text-2xl font-black text-primary">{uploadProgress}%</span>
									<Button
										type="button"
										variant="outline"
										size="sm"
										class="h-8 gap-1.5 border-destructive/40 text-destructive hover:bg-destructive hover:text-destructive-foreground transition-colors"
										onclick={cancelUpload}
									>
										<X class="h-3.5 w-3.5" />
										Cancel
									</Button>
								</div>
							</div>

							<!-- Smooth animated gradient bar -->
							<div class="relative h-3 w-full overflow-hidden rounded-full bg-muted">
								<div
									class="relative h-full rounded-full bg-gradient-to-r from-primary/80 via-primary to-primary transition-all duration-200 ease-out"
									style="width: {uploadProgress}%"
								>
									<div class="absolute inset-0 bg-white/20 animate-pulse"></div>
								</div>
							</div>

							<!-- Metrics Grid: Uploaded of Total, Speed, and Estimated Time -->
							<div class="grid grid-cols-3 gap-2 border-t border-border/60 pt-3 text-xs">
								<div class="space-y-0.5">
									<span class="text-muted-foreground block text-[11px] font-medium">Uploaded</span>
									<span class="font-mono font-bold text-foreground">
										{formatBytes(uploadedBytes)} / {formatBytes(totalBytes || file?.size || 0)}
									</span>
								</div>
								<div class="space-y-0.5 text-center">
									<span class="text-muted-foreground block text-[11px] font-medium">Upload Speed</span>
									<span class="font-mono font-bold text-foreground">
										{formatSpeed(uploadSpeed)}
									</span>
								</div>
								<div class="space-y-0.5 text-right">
									<span class="text-muted-foreground block text-[11px] font-medium">Time Remaining</span>
									<span class="font-mono font-bold text-foreground">
										{formatEta(etaSeconds)}
									</span>
								</div>
							</div>
						</div>
					{/if}

					<div class="pt-4">
						{#if isUploading}
							<Button
								type="button"
								variant="destructive"
								class="text-md h-12 w-full gap-2 font-semibold shadow-sm transition-all"
								onclick={cancelUpload}
							>
								<X class="h-5 w-5" />
								Cancel Upload
							</Button>
						{:else}
							<Button
								type="submit"
								disabled={uploadType === 'direct' && !file}
								class="text-md h-12 w-full font-semibold shadow-sm"
							>
								Upload Video
							</Button>
						{/if}
					</div>
				</form>
			{/if}
		</Card.Content>
	</Card.Root>
</div>
