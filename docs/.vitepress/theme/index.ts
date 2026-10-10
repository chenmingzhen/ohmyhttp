import DefaultTheme from 'vitepress/theme'
import { h } from 'vue'
import OutlineTitle from './OutlineTitle.vue'

export default {
  extends: DefaultTheme,
  Layout() {
    return h(DefaultTheme.Layout, null, {
      'layout-bottom': () => h(OutlineTitle),
    })
  },
}
