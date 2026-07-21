import { useEffect, useRef } from "react";

type DigitalEmployee3DProps = {
  active: boolean;
  thinking: boolean;
};

export function DigitalEmployee3D({ active, thinking }: DigitalEmployee3DProps) {
  const mountRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const mount = mountRef.current;
    if (!mount) {
      return;
    }
    let disposed = false;
    let cleanup = () => {};
    void import("three").then((THREE) => {
      if (disposed || !mountRef.current) {
        return;
      }
      const scene = new THREE.Scene();
      const camera = new THREE.PerspectiveCamera(30, 1, 0.1, 100);
      camera.position.set(0, 0.08, 5.8);

      const renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
      renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
      renderer.setSize(180, 220);
      mount.appendChild(renderer.domElement);

      const keyLight = new THREE.DirectionalLight(0xffffff, 2.6);
      keyLight.position.set(2.4, 3.4, 4.2);
      scene.add(keyLight);
      const rimLight = new THREE.DirectionalLight(0x7ff1de, 1.2);
      rimLight.position.set(-2.2, 1.8, 2.5);
      scene.add(rimLight);
      scene.add(new THREE.AmbientLight(0xb7c7e8, 1.05));

      const group = new THREE.Group();
      scene.add(group);

      const shellMaterial = new THREE.MeshStandardMaterial({ color: 0xf7fbff, roughness: 0.31, metalness: 0.1 });
      const suitMaterial = new THREE.MeshStandardMaterial({ color: 0x1f4f9a, roughness: 0.38, metalness: 0.14 });
      const trimMaterial = new THREE.MeshStandardMaterial({ color: 0x122033, roughness: 0.36, metalness: 0.18 });
      const glassMaterial = new THREE.MeshStandardMaterial({ color: 0x102033, roughness: 0.18, metalness: 0.2 });
      const cyanMaterial = new THREE.MeshStandardMaterial({ color: 0x73f3ff, emissive: 0x12b8ce, emissiveIntensity: active ? 0.9 : 0.45 });
      const greenMaterial = new THREE.MeshStandardMaterial({ color: 0x18a36d, emissive: 0x16a36d, emissiveIntensity: active ? 0.75 : 0.32 });

      const body = new THREE.Mesh(
        new THREE.CapsuleGeometry(0.58, 0.72, 14, 32),
        suitMaterial,
      );
      body.scale.set(0.96, 0.92, 0.72);
      body.position.y = -0.62;
      group.add(body);

      const torsoPanel = new THREE.Mesh(
        new THREE.BoxGeometry(0.72, 0.72, 0.08),
        new THREE.MeshStandardMaterial({ color: 0x285fb6, roughness: 0.3, metalness: 0.12 }),
      );
      torsoPanel.position.set(0, -0.58, 0.48);
      torsoPanel.rotation.x = -0.08;
      group.add(torsoPanel);

      const collar = new THREE.Mesh(
        new THREE.TorusGeometry(0.52, 0.035, 12, 80),
        trimMaterial,
      );
      collar.position.y = -0.08;
      collar.scale.set(1, 0.42, 0.2);
      collar.rotation.x = Math.PI / 2;
      group.add(collar);

      const head = new THREE.Mesh(
        new THREE.SphereGeometry(0.78, 48, 32),
        shellMaterial,
      );
      head.scale.set(1.02, 1.0, 0.86);
      head.position.y = 0.42;
      group.add(head);

      const face = new THREE.Mesh(
        new THREE.BoxGeometry(1.02, 0.46, 0.08),
        glassMaterial,
      );
      face.position.set(0, 0.46, 0.72);
      face.scale.set(1, 0.86, 0.8);
      group.add(face);

      const eyeGeometry = new THREE.BoxGeometry(0.2, 0.045, 0.035);
      const leftEye = new THREE.Mesh(eyeGeometry, cyanMaterial);
      leftEye.position.set(-0.25, 0.5, 0.79);
      const rightEye = leftEye.clone();
      rightEye.position.x = 0.26;
      group.add(leftEye, rightEye);

      const dataLine = new THREE.Mesh(
        new THREE.BoxGeometry(0.48, 0.025, 0.025),
        cyanMaterial,
      );
      dataLine.position.set(0, 0.33, 0.79);
      group.add(dataLine);

      const earGeometry = new THREE.SphereGeometry(0.18, 24, 16);
      const leftEar = new THREE.Mesh(earGeometry, trimMaterial);
      leftEar.scale.set(0.58, 1.18, 0.64);
      leftEar.position.set(-0.76, 0.44, 0.02);
      const rightEar = leftEar.clone();
      rightEar.position.x = 0.82;
      group.add(leftEar, rightEar);

      const headsetBand = new THREE.Mesh(
        new THREE.TorusGeometry(0.78, 0.022, 10, 80, Math.PI),
        trimMaterial,
      );
      headsetBand.position.set(0, 0.57, -0.02);
      headsetBand.rotation.x = Math.PI / 2;
      headsetBand.rotation.z = Math.PI;
      group.add(headsetBand);

      const micArm = new THREE.Mesh(
        new THREE.CylinderGeometry(0.014, 0.014, 0.34, 10),
        trimMaterial,
      );
      micArm.position.set(0.56, 0.2, 0.55);
      micArm.rotation.z = 0.82;
      group.add(micArm);

      const micDot = new THREE.Mesh(new THREE.SphereGeometry(0.045, 16, 10), greenMaterial);
      micDot.position.set(0.69, 0.06, 0.63);
      group.add(micDot);

      const shoulderGeometry = new THREE.CapsuleGeometry(0.11, 0.62, 10, 18);
      const leftArm = new THREE.Mesh(shoulderGeometry, suitMaterial);
      leftArm.position.set(-0.68, -0.5, 0.02);
      leftArm.rotation.z = 0.62;
      const rightArm = leftArm.clone();
      rightArm.position.x = 0.72;
      rightArm.rotation.z = -0.62;
      group.add(leftArm, rightArm);

      const core = new THREE.Mesh(new THREE.SphereGeometry(0.095, 24, 16), greenMaterial);
      core.position.set(0, -0.48, 0.55);
      group.add(core);

      const badge = new THREE.Mesh(
        new THREE.TorusGeometry(0.18, 0.012, 8, 48),
        greenMaterial,
      );
      badge.position.set(0, -0.48, 0.56);
      badge.rotation.x = Math.PI / 2;
      group.add(badge);

      const halo = new THREE.Mesh(
        new THREE.TorusGeometry(0.95, 0.018, 12, 96),
        cyanMaterial,
      );
      halo.position.y = -1.19;
      halo.rotation.x = Math.PI / 2;
      group.add(halo);

      const shadow = new THREE.Mesh(
        new THREE.CircleGeometry(0.96, 64),
        new THREE.MeshBasicMaterial({ color: 0x7fa6df, transparent: true, opacity: 0.22 }),
      );
      shadow.position.y = -1.34;
      shadow.rotation.x = -Math.PI / 2;
      group.add(shadow);

      let frame = 0;
      let animationId = 0;
      const animate = () => {
        frame += 0.018;
        group.rotation.y = Math.sin(frame) * 0.12;
        group.position.y = Math.sin(frame * 1.5) * 0.035;
        head.rotation.z = Math.sin(frame * 1.1) * 0.018;
        leftArm.rotation.z = 0.62 + Math.sin(frame * 1.7) * 0.035;
        rightArm.rotation.z = -0.62 + Math.cos(frame * 1.7) * 0.035;
        core.scale.setScalar(1 + Math.sin(frame * 4.4) * (thinking ? 0.22 : 0.08));
        micDot.scale.setScalar(1 + Math.cos(frame * 4.1) * (thinking ? 0.15 : 0.05));
        halo.rotation.z += thinking ? 0.028 : 0.01;
        const pulse = active || thinking ? 0.06 : 0.025;
        halo.scale.setScalar(1 + Math.sin(frame * 4) * pulse);
        renderer.render(scene, camera);
        animationId = requestAnimationFrame(animate);
      };
      animate();

      const resizeObserver = new ResizeObserver((entries) => {
        const rect = entries[0]?.contentRect;
        if (!rect) {
          return;
        }
        const width = Math.max(150, Math.floor(rect.width));
        const height = Math.max(180, Math.floor(rect.height));
        renderer.setSize(width, height);
        camera.aspect = width / height;
        camera.updateProjectionMatrix();
      });
      resizeObserver.observe(mount);

      cleanup = () => {
        cancelAnimationFrame(animationId);
        resizeObserver.disconnect();
        if (renderer.domElement.parentElement === mount) {
          mount.removeChild(renderer.domElement);
        }
        renderer.dispose();
        scene.traverse((object) => {
          if (object instanceof THREE.Mesh) {
            object.geometry.dispose();
            const material = object.material;
            if (Array.isArray(material)) {
              material.forEach((item) => item.dispose());
            } else {
              material.dispose();
            }
          }
        });
      };
    });

    return () => {
      disposed = true;
      cleanup();
    };
  }, [active, thinking]);

  return <div className="employee-3d-stage" ref={mountRef} aria-label="Digital employee 3D avatar" />;
}
