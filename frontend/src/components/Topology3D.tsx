import { useEffect, useMemo } from 'react'
import { Canvas } from '@react-three/fiber'
import { OrbitControls, Text, Grid } from '@react-three/drei'
import { EffectComposer, Bloom } from '@react-three/postprocessing'
import * as THREE from 'three'
import { useStore } from '@/store/useStore'
import { fetchTopology } from '@/utils/api'
import SubstationNode from './SubstationNode'
import FeederLine from './FeederLine'

const LINE_NAMES = ['1号线', '2号线', '3号线']
const LINE_OFFSETS = [-18, 0, 18]
const STATIONS_PER_LINE = 20

function generateDemoTopology() {
  const nodes: ReturnType<typeof useStore.getState>['topologyNodes'] = []
  const edges: ReturnType<typeof useStore.getState>['topologyEdges'] = []

  for (let lineIdx = 0; lineIdx < 3; lineIdx++) {
    const lineId = `line-${lineIdx + 1}`
    const lineName = LINE_NAMES[lineIdx]
    const xOffset = LINE_OFFSETS[lineIdx]

    for (let i = 0; i < STATIONS_PER_LINE; i++) {
      const id = `ss-${lineIdx + 1}-${i + 1}`
      const loadRate = Math.random() * 100
      nodes.push({
        id,
        name: `${lineName}-${i + 1}站`,
        type: 'substation',
        lineId,
        lineName,
        x: xOffset,
        y: 1.5,
        z: -STATIONS_PER_LINE * 2.2 + i * 4.4,
        loadRate,
        voltage: 33 + Math.random() * 2 - 1,
        current: 200 + Math.random() * 600,
        power: 6 + Math.random() * 20,
        temperature: 35 + Math.random() * 30,
        status: loadRate > 80 ? 'fault' : loadRate > 60 ? 'warning' : 'normal',
      })

      if (i > 0) {
        const prevId = `ss-${lineIdx + 1}-${i}`
        edges.push({
          id: `feeder-${lineIdx + 1}-${i}`,
          source: prevId,
          target: id,
          type: 'feeder',
          loadRate: Math.random() * 100,
          power: 5 + Math.random() * 15,
          fault: Math.random() < 0.05,
        })
      }
    }
  }

  edges.push(
    {
      id: 'tie-1-2',
      source: 'ss-1-10',
      target: 'ss-2-10',
      type: 'tie',
      loadRate: 25,
      power: 3,
      fault: false,
    },
    {
      id: 'tie-2-3',
      source: 'ss-2-15',
      target: 'ss-3-15',
      type: 'tie',
      loadRate: 18,
      power: 2,
      fault: false,
    }
  )

  return { nodes, edges }
}

function Scene() {
  const nodes = useStore((s) => s.topologyNodes)
  const edges = useStore((s) => s.topologyEdges)
  const setSelectedDevice = useStore((s) => s.setSelectedDevice)

  const nodeMap = useMemo(() => {
    const map = new Map<string, (typeof nodes)[0]>()
    nodes.forEach((n) => map.set(n.id, n))
    return map
  }, [nodes])

  return (
    <>
      <ambientLight intensity={0.3} />
      <directionalLight position={[30, 50, 20]} intensity={0.6} color="#b0d0ff" />
      <directionalLight position={[-20, 30, -10]} intensity={0.2} color="#00D4FF" />

      {LINE_OFFSETS.map((offset, idx) => (
        <Text
          key={`line-label-${idx}`}
          position={[offset, 5, STATIONS_PER_LINE * 2.2 - 2]}
          fontSize={1}
          color="#00D4FF"
          anchorX="center"
          anchorY="middle"
          font={undefined}
        >
          {LINE_NAMES[idx]}
        </Text>
      ))}

      <Grid
        position={[0, -0.5, 0]}
        args={[120, 120]}
        cellSize={2}
        cellThickness={0.5}
        cellColor="#1a3a5c"
        sectionSize={10}
        sectionThickness={1}
        sectionColor="#1a4a6c"
        fadeDistance={80}
        infiniteGrid={false}
      />

      {nodes.map((node) => (
        <SubstationNode
          key={node.id}
          id={node.id}
          name={node.name}
          position={[node.x, node.y, node.z]}
          loadRate={node.loadRate}
          status={node.status}
          onClick={setSelectedDevice}
        />
      ))}

      {edges.map((edge) => {
        const sourceNode = nodeMap.get(edge.source)
        const targetNode = nodeMap.get(edge.target)
        if (!sourceNode || !targetNode) return null

        return (
          <FeederLine
            key={edge.id}
            sourcePos={[sourceNode.x, sourceNode.y, sourceNode.z]}
            targetPos={[targetNode.x, targetNode.y, targetNode.z]}
            loadRate={edge.loadRate}
            fault={edge.fault}
          />
        )
      })}

      <OrbitControls
        makeDefault
        enableDamping
        dampingFactor={0.05}
        minDistance={10}
        maxDistance={120}
        maxPolarAngle={Math.PI / 2.1}
        target={[0, 0, 0]}
      />
    </>
  )
}

export default function Topology3D() {
  const setTopology = useStore((s) => s.setTopology)
  const nodes = useStore((s) => s.topologyNodes)

  useEffect(() => {
    fetchTopology()
      .then((data) => {
        setTopology(data.nodes, data.edges)
      })
      .catch(() => {
        const demo = generateDemoTopology()
        setTopology(demo.nodes, demo.edges)
      })
  }, [setTopology])

  return (
    <Canvas
      camera={{
        position: [40, 35, 40],
        fov: 50,
        near: 0.1,
        far: 500,
      }}
      gl={{
        antialias: true,
        toneMapping: THREE.ACESFilmicToneMapping,
        toneMappingExposure: 1.2,
      }}
      style={{ background: '#0A1628' }}
      onPointerMissed={() => useStore.getState().setSelectedDevice(null)}
    >
      <Scene />
      <EffectComposer>
        <Bloom
          intensity={0.8}
          luminanceThreshold={0.2}
          luminanceSmoothing={0.9}
          mipmapBlur
        />
      </EffectComposer>
    </Canvas>
  )
}
