<script lang="ts">
    import { createEventDispatcher } from 'svelte';
    import { CloudUpload, CircleCheckBig, Loader, CircleAlert } from 'lucide-svelte';
    import Tesseract from 'tesseract.js';
    import pdfjsWorkerUrl from 'pdfjs-dist/build/pdf.worker.min.mjs?url';

    const dispatch = createEventDispatcher();

    let isDragging = false;
    let file: File | null = null;
    let previewUrl: string | null = null;
    let status: 'idle' | 'processing' | 'success' | 'error' = 'idle';
    let progress = 0;
    let errorMsg = '';

    function handleDrag(e: DragEvent) {
        e.preventDefault();
        e.stopPropagation();
        if (e.type === 'dragenter' || e.type === 'dragover') {
            isDragging = true;
        } else if (e.type === 'dragleave') {
            isDragging = false;
        }
    }

    async function handleDrop(e: DragEvent) {
        e.preventDefault();
        e.stopPropagation();
        isDragging = false;

        const droppedFiles = e.dataTransfer?.files;
        if (droppedFiles && droppedFiles.length > 0) {
            await processFile(droppedFiles[0]);
        }
    }

    async function handleFileSelect(e: Event) {
        const input = e.target as HTMLInputElement;
        if (input.files && input.files.length > 0) {
            await processFile(input.files[0]);
        }
    }

    async function processFile(selectedFile: File) {
        let imageFile = selectedFile;

        if (selectedFile.type === 'application/pdf') {
            status = 'processing';
            errorMsg = '';
            try {
                // Dynamically import to keep bundle small
                const pdfjsLib = await import('pdfjs-dist');
                // Use the bundled worker URL provided by Vite instead of CDN
                pdfjsLib.GlobalWorkerOptions.workerSrc = pdfjsWorkerUrl;
                
                const arrayBuffer = await selectedFile.arrayBuffer();
                const pdf = await pdfjsLib.getDocument(arrayBuffer).promise;
                const page = await pdf.getPage(1); // Render first page only
                const viewport = page.getViewport({ scale: 2.0 }); // High-res for OCR
                
                const canvasElem = document.createElement('canvas');
                const context = canvasElem.getContext('2d');
                canvasElem.height = viewport.height;
                canvasElem.width = viewport.width;
                
                if (context) {
                    await page.render({ canvasContext: context, viewport: viewport } as any).promise;
                    const blob = await new Promise<Blob | null>(resolve => canvasElem.toBlob(resolve, 'image/jpeg', 0.9));
                    if (!blob) throw new Error("Canvas toBlob failed");
                    imageFile = new File([blob], selectedFile.name.replace(/\.pdf$/i, '.jpg'), { type: 'image/jpeg' });
                }
            } catch (err: any) {
                status = 'error';
                errorMsg = 'PDF conversion failed: ' + (err.message || 'Unknown error');
                return;
            }
        } else if (!selectedFile.type.startsWith('image/')) {
            status = 'error';
            errorMsg = 'Only image or PDF files are supported for OCR currently.';
            return;
        }

        file = imageFile;
        if (previewUrl) URL.revokeObjectURL(previewUrl);
        previewUrl = URL.createObjectURL(file);
        status = 'processing';
        progress = 0;
        errorMsg = '';

        try {
            const worker = await Tesseract.createWorker('eng', 1, {
                logger: (m: any) => {
                    if (m.status === 'recognizing text') {
                        progress = Math.round(m.progress * 100);
                    }
                }
            });

            const { data: { text } } = await worker.recognize(file);
            await worker.terminate();

            status = 'success';
            dispatch('extracted', { text, file });
        } catch (err: any) {
            status = 'error';
            errorMsg = err.message || 'OCR extraction failed.';
            console.error(err);
        }
    }
</script>

<div 
    class="w-full relative rounded-2xl border-2 border-dashed transition-all duration-300 flex flex-col items-center justify-center min-h-[250px] p-8 overflow-hidden group
    {isDragging ? 'border-primary bg-primary/10' : 'border-surfaceAcc hover:border-primary/50 bg-background/50 hover:bg-surface/50'}
    {status === 'success' ? 'border-success bg-success/5' : ''}
    {status === 'error' ? 'border-danger bg-danger/5' : ''}"
    on:dragenter={handleDrag}
    on:dragleave={handleDrag}
    on:dragover={handleDrag}
    on:drop={handleDrop}
>
    
    <input 
        type="file" 
        accept="image/*,application/pdf" 
        class="absolute inset-0 w-full h-full opacity-0 cursor-pointer z-10"
        on:change={handleFileSelect}
        disabled={status === 'processing'}
    />

    {#if status === 'idle'}
        <div class="bg-surfaceAcc/30 p-4 rounded-full text-primary mb-4 group-hover:scale-110 transition-transform">
            <CloudUpload size={32} />
        </div>
        <h3 class="text-lg font-bold">Upload Document</h3>
        <p class="text-sm text-gray-400 mt-2 text-center max-w-sm">
            Drag and drop your scanned invoice or PDF here, or click to browse. The application will automatically extract the generic data fields via local OCR.
        </p>
    
    {:else if status === 'processing'}
        <div class="relative w-full max-w-xs text-center z-20">
            <Loader size={40} class="animate-spin text-primary mx-auto mb-4" />
            <h3 class="text-lg font-bold animate-pulse text-primary">Scanning Document...</h3>
            <p class="text-sm text-gray-400 mt-2">Running Neural OCR Engines</p>
            
            <div class="w-full h-2 bg-surfaceAcc rounded-full mt-6 overflow-hidden">
                <div class="h-full bg-primary rounded-full transition-all duration-300 ease-out" style="width: {progress}%"></div>
            </div>
            <div class="text-xs text-gray-500 mt-2 font-mono">{progress}% Complete</div>
        </div>
        
    {:else if status === 'success'}
        <div class="bg-success/20 p-4 rounded-full text-success mb-4 relative z-20">
            <CircleCheckBig size={32} />
        </div>
        <h3 class="text-lg font-bold text-success relative z-20">Extraction Complete!</h3>
        <p class="text-sm text-gray-400 mt-2 relative z-20">The data has been populated in the form below.</p>
        
        <button 
            class="mt-6 px-4 py-2 bg-surface text-sm border border-surfaceAcc rounded-lg hover:bg-surfaceAcc/50 transition-colors z-20 relative cursor-pointer"
            on:click={() => { status = 'idle'; file = null; if (previewUrl) URL.revokeObjectURL(previewUrl); previewUrl = null; dispatch('reset'); }}
        >
            Scan Another Document
        </button>

        {#if previewUrl}
            <div class="absolute inset-0 opacity-10 blur-[2px] pointer-events-none">
                <img src={previewUrl} alt="Preview" class="w-full h-full object-cover">
            </div>
        {/if}

    {:else if status === 'error'}
        <div class="bg-danger/20 p-4 rounded-full text-danger mb-4 relative z-20">
            <CircleAlert size={32} />
        </div>
        <h3 class="text-lg font-bold text-danger relative z-20">Processing Failed</h3>
        <p class="text-sm text-gray-400 mt-2 relative z-20">{errorMsg}</p>
        <button 
            class="mt-6 px-4 py-2 bg-surface text-sm border border-danger/50 text-danger rounded-lg hover:bg-danger/10 transition-colors z-20 relative cursor-pointer"
            on:click={() => status = 'idle'}
        >
            Try Again
        </button>
    {/if}

</div>
