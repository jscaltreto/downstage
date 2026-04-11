<script setup lang="ts">
import { ref, watch, onMounted } from 'vue';

const props = defineProps<{
  html: string;
}>();

const iframe = ref<HTMLIFrameElement | null>(null);

function updateIframe() {
  if (!iframe.value) return;
  
  const scrollEl = iframe.value.contentDocument?.scrollingElement || iframe.value.contentDocument?.documentElement;
  const savedScroll = scrollEl?.scrollTop || 0;

  iframe.value.srcdoc = props.html;

  const onLoad = () => {
    const el = iframe.value?.contentDocument?.scrollingElement || iframe.value?.contentDocument?.documentElement;
    if (el) el.scrollTop = savedScroll;
    iframe.value?.removeEventListener("load", onLoad);
  };
  iframe.value.addEventListener("load", onLoad);
}

watch(() => props.html, updateIframe);
onMounted(updateIframe);

defineExpose({ 
  get iframeEl() { return iframe.value } 
});
</script>

<template>
  <iframe 
    ref="iframe" 
    class="w-full h-full bg-white border-none"
    sandbox="allow-same-origin"
    title="Live Preview"
  ></iframe>
</template>
