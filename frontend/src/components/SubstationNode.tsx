import { useRef, useState } from 'react'
import { useFrame } from '@react-three/fiber'
import { Text } from '@react-three/drei'
import * as THREE from 'three'
import { useStore, getLoadRateColor } from '@/store/useStore'

interface SubstationNodeProps {
  id: string
  name: string
  position: [number, number, number]
  loadRate: number
  status: 'normal' | 'warning' | 'fault'
  onClick: (id: string) => void
}

export default function SubstationNode({
  id,
  name,
  position,
  loadRate,
  status,
  onClick,
}: SubstationNodeProps) {
  const meshRef = useRef<THREE.Mesh>(null)
  const edgesRef = useRef<THREE.LineSegments>(null)
  const [hovered, setHovered] = useState(false)
  const selectedDevice = useStore((s) => s.selectedDevice)
  const isSelected = selectedDevice === id

  const color = getLoadRateColor(loadRate)
  const threeColor = new THREE.Color(color)

  useFrame((state) => {
    if (meshRef.current) {
      meshRef.current.position.y =
        position[1] + Math.sin(state.clock.elapsedTime * 0.8 + position[0] * 0.5) * 0.08

      const targetScale = hovered || isSelected ? 1.15 : 1
      meshRef.current.scale.lerp(
        new THREE.Vector3(targetScale, targetScale, targetScale),
        0.1
      )

      const pulseMultiplier = status === 'fault'
        ? 0.5 + Math.sin(state.clock.elapsedTime * 4) * 0.5
        : 1
      const emissiveIntensity = (hovered || isSelected ? 1.2 : 0.5) * pulseMultiplier
      const material = meshRef.current.material as THREE.MeshStandardMaterial
      material.emissiveIntensity = emissiveIntensity
    }

    if (edgesRef.current) {
      edgesRef.current.position.y =
        position[1] + Math.sin(state.clock.elapsedTime * 0.8 + position[0] * 0.5) * 0.08

      const edgeMaterial = edgesRef.current.material as THREE.LineBasicMaterial
      edgeMaterial.opacity = 0.6 + (hovered || isSelected ? 0.4 : 0)
    }
  })

  const geometry = new THREE.BoxGeometry(1.5, 1.5, 1.5)
  const edgesGeometry = new THREE.EdgesGeometry(geometry)

  return (
    <group position={[position[0], 0, position[2]]}>
      <mesh
        ref={meshRef}
        position={[0, position[1], 0]}
        geometry={geometry}
        onClick={(e) => {
          e.stopPropagation()
          onClick(id)
        }}
        onPointerOver={(e) => {
          e.stopPropagation()
          setHovered(true)
          document.body.style.cursor = 'pointer'
        }}
        onPointerOut={() => {
          setHovered(false)
          document.body.style.cursor = 'default'
        }}
      >
        <meshStandardMaterial
          color={threeColor}
          emissive={threeColor}
          emissiveIntensity={0.5}
          transparent
          opacity={0.85}
          roughness={0.3}
          metalness={0.7}
        />
        <lineSegments ref={edgesRef} position={[0, 0, 0]} geometry={edgesGeometry}>
          <lineBasicMaterial
            color={threeColor}
            transparent
            opacity={0.8}
          />
        </lineSegments>
      </mesh>

      <Text
        position={[0, position[1] + 1.6, 0]}
        fontSize={0.5}
        color="#ffffff"
        anchorX="center"
        anchorY="bottom"
        font={undefined}
        outlineWidth={0.02}
        outlineColor="#000000"
      >
        {name}
      </Text>

      <Text
        position={[0, position[1] - 1.3, 0]}
        fontSize={0.35}
        color={color}
        anchorX="center"
        anchorY="top"
        font={undefined}
      >
        {`${loadRate.toFixed(1)}%`}
      </Text>

      {isSelected && (
        <pointLight
          position={[0, position[1], 0]}
          color={color}
          intensity={2}
          distance={6}
        />
      )}
    </group>
  )
}
