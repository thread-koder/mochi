import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import {
  AxisPointerComponent,
  GridComponent,
  TooltipComponent,
} from 'echarts/components'

export default defineNuxtPlugin(() => {
  use([
    CanvasRenderer,
    LineChart,
    GridComponent,
    TooltipComponent,
    AxisPointerComponent,
  ])
})
