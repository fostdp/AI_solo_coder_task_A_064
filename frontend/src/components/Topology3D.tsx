import Power3DViewer from './Power3DViewer'
import DeviceDashboard from './DeviceDashboard'
import { useStore } from '@/store/useStore'

export default function Topology3D() {
  const selectedDevice = useStore((s) => s.selectedDevice)
  return (
    <div className="relative w-full h-full">
      <Power3DViewer />
      {selectedDevice && <DeviceDashboard />}
    </div>
  )
}
