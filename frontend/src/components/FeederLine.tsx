import { useRef, useMemo } from 'react'
import { useFrame } from '@react-three/fiber'
import * as THREE from 'three'
import { getLoadRateColor } from '@/store/useStore'

interface FeederLineProps {
  sourcePos: [number, number, number]
  targetPos: [number, number, number]
  loadRate: number
  fault: boolean
}

export default function FeederLine({
  sourcePos,
  targetPos,
  loadRate,
  fault,
}: FeederLineProps) {
  const materialRef = useRef<THREE.MeshBasicMaterial>(null)
  const glowMaterialRef = useRef<THREE.MeshBasicMaterial>(null)

  const color = fault ? '#FF3344' : getLoadRateColor(loadRate)
  const threeColor = new THREE.Color(color)

  const { tubeGeometry, glowTubeGeometry } = useMemo(() => {
    const curve = new THREE.LineCurve3(
      new THREE.Vector3(...sourcePos),
      new THREE.Vector3(...targetPos)
    )
    const tubeGeo = new THREE.TubeGeometry(curve, 1, fault ? 0.08 : 0.05, 6, false)
    const glowTubeGeo = new THREE.TubeGeometry(curve, 1, fault ? 0.15 : 0.1, 6, false)
    return { tubeGeometry: tubeGeo, glowTubeGeometry: glowTubeGeo }
  }, [sourcePos[0], sourcePos[1], sourcePos[2], targetPos[0], targetPos[1], targetPos[2], fault])

  useFrame((state) => {
    if (fault) {
      const pulse = 0.3 + Math.sin(state.clock.elapsedTime * 6) * 0.7
      if (materialRef.current) {
        materialRef.current.opacity = pulse
      }
      if (glowMaterialRef.current) {
        glowMaterialRef.current.opacity = pulse * 0.3
      }
    }
  })

  const isLowLoad = loadRate < 30

  return (
    <group>
      <mesh geometry={glowTubeGeometry}>
        <meshBasicMaterial
          ref={glowMaterialRef}
          color={threeColor}
          transparent
          opacity={0.15}
          side={THREE.DoubleSide}
        />
      </mesh>

      <mesh geometry={tubeGeometry}>
        <meshBasicMaterial
          ref={materialRef}
          color={threeColor}
          transparent
          opacity={isLowLoad ? 0.3 : fault ? 0.8 : 0.7}
          side={THREE.DoubleSide}
        />
      </mesh>
    </group>
  )
}
