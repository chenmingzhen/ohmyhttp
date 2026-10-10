<script setup lang="ts">
import { onContentUpdated } from 'vitepress'
import { onMounted } from 'vue'

function pageTitle() {
  const heading = document.querySelector('.vp-doc h1')
  if (!heading) return ''
  const copy = heading.cloneNode(true) as HTMLElement
  copy.querySelectorAll('.header-anchor').forEach((node) => node.remove())
  return copy.textContent?.trim() ?? ''
}

function applyPageTitle() {
  if (typeof document === 'undefined') return
  const title = pageTitle()
  if (!title) return
  document
    .querySelectorAll(
      '.VPDocAsideOutline .outline-title, .VPLocalNavOutlineDropdown .menu-text',
    )
    .forEach((node) => {
      node.textContent = title
    })
}

onContentUpdated(() => requestAnimationFrame(applyPageTitle))
onMounted(applyPageTitle)
</script>

<template><span hidden /></template>
