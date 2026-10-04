<script lang="ts">
	import { onMount } from 'svelte';
	import { fetchApi } from '#lib/api';
	import { userState } from '#lib/user.svelte';
	import { UploadCloud, CheckCircle2, ArrowLeft, Loader2 } from 'lucide-svelte';

	let file: File | null = $state(null);
	let title = $state('');
	let description = $state('');
	let category = $state('General');
	let visibility = $state('public');
	
	let isUploading = $state(false);
	let uploadProgress = $state(0);
	let uploadStatus = $state(''); // 'idle', 'uploading', 'processing', 'success', 'error'
	let errorMessage = $state('');
	let uploadInput: HTMLInputElement;

	onMount(() => {
		if (!userState.user && userState.isLoaded) {
			window.location.href = '/login';
		}
	});

	function handleFileSelect(e: Event) {
		const target = e.target as HTMLInputElement;
		if (target.files && target.files.length > 0) {
			file = target.files[0];
			if (!title) {
				// Remove extension for default title
				title = file.name.replace(/\.[^/.]+$/, "");
			}
		}
	}

	function handleDragOver(e: DragEvent) {
		e.preventDefault();
	}

	function handleDrop(e: DragEvent) {
		e.preventDefault();
		if (e.dataTransfer?.files && e.dataTransfer.files.length > 0) {
			file = e.dataTransfer.files[0];
			if (!title) {
				title = file.name.replace(/\.[^/.]+$/, "");
			}
		}
	}

	async function startUpload() {
		if (!file) {
			errorMessage = 'Please select a video file.';
			return;
		}

		isUploading = true;
		uploadStatus = 'uploading';
		errorMessage = '';
		uploadProgress = 0;

		try {
			// 1. Get direct upload URL from backend
			const urlRes = await fetchApi('/videos/get-upload-url');
			if (!urlRes.url) {
				throw new Error("Could not retrieve upload URL.");
			}
			const uploadUrl = urlRes.url;

			// 2. Direct upload to Streamtape
			const formData = new FormData();
			formData.append('file1', file);

			// We use XMLHttpRequest to track upload progress accurately
			const videoId = await new Promise<string>((resolve, reject) => {
				const xhr = new XMLHttpRequest();
				
				xhr.upload.onprogress = (e) => {
					if (e.lengthComputable) {
						uploadProgress = Math.round((e.loaded / e.total) * 100);
					}
				};

				xhr.onload = () => {
					if (xhr.status >= 200 && xhr.status < 300) {
						try {
							const response = JSON.parse(xhr.responseText);
							if (response && response.result && response.result.id) {
								resolve(response.result.id);
							} else {
								reject(new Error("Invalid response format from upload server."));
							}
						} catch (err) {
							reject(new Error("Failed to parse upload response."));
						}
					} else {
						reject(new Error(`Upload failed with status ${xhr.status}`));
					}
				};

				xhr.onerror = () => reject(new Error("Network error during upload."));
				xhr.open("POST", uploadUrl);
				xhr.send(formData);
			});

			uploadStatus = 'processing';

			// 3. Create the database record
			const recordFormData = new FormData();
			recordFormData.append('title', title);
			recordFormData.append('description', description);
			recordFormData.append('category', category);
			recordFormData.append('visibility', visibility);
			recordFormData.append('videoId', videoId);
			// Optional: custom thumbnail can be appended here

			await fetchApi('/videos/create-record', {
				method: 'POST',
				body: recordFormData
			});

			uploadStatus = 'success';
		} catch (err: any) {
			uploadStatus = 'error';
			errorMessage = err.message || 'An error occurred during upload.';
		} finally {
			isUploading = false;
		}
	}
</script>

<svelte:head>
	<title>Upload Video - Aurahub</title>
</svelte:head>

<div class="max-w-3xl mx-auto space-y-6">
	<div class="flex items-center gap-4">
		<a href="/dashboard" class="p-2 rounded-full hover:bg-muted transition-colors">
			<ArrowLeft class="h-5 w-5" />
		</a>
		<div>
			<h1 class="text-2xl font-bold tracking-tight">Upload Video</h1>
			<p class="text-sm text-muted-foreground">Share your content with the Aurahub community.</p>
		</div>
	</div>

	{#if uploadStatus === 'success'}
		<div class="bg-card border rounded-xl p-12 flex flex-col items-center justify-center text-center space-y-4 shadow-sm">
			<div class="bg-green-100 dark:bg-green-900/30 text-green-600 p-4 rounded-full">
				<CheckCircle2 class="h-12 w-12" />
			</div>
			<h2 class="text-2xl font-bold tracking-tight">Upload Complete!</h2>
			<p class="text-muted-foreground">Your video has been successfully uploaded and is now processing.</p>
			<a href="/dashboard" class="inline-flex h-10 items-center justify-center rounded-md bg-primary px-8 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 mt-4">
				Return to Dashboard
			</a>
		</div>
	{:else}
		<div class="grid grid-cols-1 md:grid-cols-2 gap-8">
			<!-- Left side: File selection -->
			<div class="space-y-4">
				<div 
					class="border-2 border-dashed rounded-xl p-8 flex flex-col items-center justify-center text-center transition-colors {file ? 'border-primary bg-primary/5' : 'border-border hover:border-primary/50 bg-card'} h-[300px]"
					ondragover={handleDragOver}
					ondrop={handleDrop}
				>
					{#if file}
						<div class="bg-primary/10 text-primary p-4 rounded-full mb-4">
							<CheckCircle2 class="h-8 w-8" />
						</div>
						<p class="font-medium text-primary truncate max-w-full px-4">{file.name}</p>
						<p class="text-xs text-muted-foreground mt-1">{(file.size / (1024 * 1024)).toFixed(2)} MB</p>
						<button 
							class="text-xs font-medium text-red-500 hover:text-red-600 mt-4"
							onclick={() => file = null}
							disabled={isUploading}
						>
							Remove File
						</button>
					{:else}
						<div class="bg-muted p-4 rounded-full mb-4">
							<UploadCloud class="h-8 w-8 text-muted-foreground" />
						</div>
						<h3 class="font-medium mb-1">Drag and drop video</h3>
						<p class="text-sm text-muted-foreground mb-4">or click to browse files</p>
						<button 
							class="inline-flex h-9 items-center justify-center rounded-md bg-secondary text-secondary-foreground px-4 text-sm font-medium hover:bg-secondary/80"
							onclick={() => uploadInput.click()}
						>
							Select File
						</button>
						<input 
							type="file" 
							accept="video/*" 
							class="hidden" 
							bind:this={uploadInput} 
							onchange={handleFileSelect} 
						/>
					{/if}
				</div>

				{#if isUploading}
					<div class="bg-card border rounded-lg p-4 space-y-3">
						<div class="flex justify-between text-sm font-medium">
							<span>{uploadStatus === 'processing' ? 'Finalizing...' : 'Uploading...'}</span>
							<span>{uploadProgress}%</span>
						</div>
						<div class="w-full bg-secondary rounded-full h-2 overflow-hidden">
							<div class="bg-primary h-full transition-all duration-300" style="width: {uploadProgress}%"></div>
						</div>
					</div>
				{/if}

				{#if errorMessage}
					<div class="bg-red-100 dark:bg-red-900/30 text-red-600 dark:text-red-400 p-3 rounded-lg text-sm font-medium">
						{errorMessage}
					</div>
				{/if}
			</div>

			<!-- Right side: Details -->
			<div class="space-y-4">
				<div class="bg-card border rounded-xl p-6 space-y-4">
					<h3 class="font-semibold text-lg tracking-tight">Video Details</h3>
					
					<div class="space-y-2">
						<label for="title" class="text-sm font-medium">Title</label>
						<input
							id="title"
							bind:value={title}
							disabled={isUploading}
							class="flex h-10 w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
							placeholder="Enter a captivating title"
						/>
					</div>

					<div class="space-y-2">
						<label for="description" class="text-sm font-medium">Description</label>
						<textarea
							id="description"
							bind:value={description}
							disabled={isUploading}
							rows="4"
							class="flex w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
							placeholder="Tell viewers about your video"
						></textarea>
					</div>

					<div class="grid grid-cols-2 gap-4">
						<div class="space-y-2">
							<label for="category" class="text-sm font-medium">Category</label>
							<select
								id="category"
								bind:value={category}
								disabled={isUploading}
								class="flex h-10 w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
							>
								<option value="General">General</option>
								<option value="Gaming">Gaming</option>
								<option value="Music">Music</option>
								<option value="Education">Education</option>
								<option value="Entertainment">Entertainment</option>
							</select>
						</div>
						<div class="space-y-2">
							<label for="visibility" class="text-sm font-medium">Visibility</label>
							<select
								id="visibility"
								bind:value={visibility}
								disabled={isUploading}
								class="flex h-10 w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
							>
								<option value="public">Public</option>
								<option value="unlisted">Unlisted</option>
								<option value="private">Private</option>
							</select>
						</div>
					</div>

					<div class="pt-4">
						<button
							onclick={startUpload}
							disabled={isUploading || !file || !title}
							class="inline-flex w-full items-center justify-center rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground shadow transition-colors hover:bg-primary/90 disabled:opacity-50 disabled:pointer-events-none"
						>
							{#if isUploading}
								<Loader2 class="mr-2 h-4 w-4 animate-spin" />
								Processing...
							{:else}
								Upload Video
							{/if}
						</button>
					</div>
				</div>
			</div>
		</div>
	{/if}
</div>
