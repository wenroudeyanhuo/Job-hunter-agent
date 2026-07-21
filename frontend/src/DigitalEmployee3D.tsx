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
      const camera = new THREE.PerspectiveCamera(32, 1, 0.1, 100);
      camera.position.set(0, 0.12, 5.6);

      const renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
      renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
      renderer.setSize(180, 220);
      mount.appendChild(renderer.domElement);

      const keyLight = new THREE.DirectionalLight(0xffffff, 2.5);
      keyLight.position.set(2.4, 3.2, 4);
      scene.add(keyLight);
      const rimLight = new THREE.DirectionalLight(0x88f7d2, 1.4);
      rimLight.position.set(-2.2, 1.8, 2.5);
      scene.add(rimLight);
      scene.add(new THREE.AmbientLight(0xa9c7ff, 1.15));

      const group = new THREE.Group();
      scene.add(group);

      const shellMaterial = new THREE.MeshStandardMaterial({ color: 0xf8fbff, roughness: 0.36, metalness: 0.08 });
      const blueMaterial = new THREE.MeshStandardMaterial({ color: 0x4b8df7, roughness: 0.34, metalness: 0.1 });
      const darkMaterial = new THREE.MeshStandardMaterial({ color: 0x18212f, roughness: 0.4 });
      const cheekMaterial = new THREE.MeshStandardMaterial({ color: 0xff9fb3, roughness: 0.58 });
      const glowMaterial = new THREE.MeshStandardMaterial({ color: 0x18a36d, emissive: 0x16a36d, emissiveIntensity: active ? 0.7 : 0.28 });

      const body = new THREE.Mesh(
        new THREE.SphereGeometry(0.72, 42, 28),
        blueMaterial,
      );
      body.scale.set(0.82, 0.92, 0.7);
      body.position.y = -0.54;
      group.add(body);

      const head = new THREE.Mesh(
        new THREE.SphereGeometry(0.84, 48, 32),
        shellMaterial,
      );
      head.scale.set(1.05, 0.92, 0.88);
      head.position.y = 0.42;
      group.add(head);

      const face = new THREE.Mesh(
        new THREE.BoxGeometry(1.05, 0.38, 0.08),
        new THREE.MeshStandardMaterial({ color: 0x1f2937, roughness: 0.32, metalness: 0.12 }),
      );
      face.position.set(0, 0.46, 0.72);
      face.scale.set(1, 1, 0.8);
      group.add(face);

      const eyeGeometry = new THREE.SphereGeometry(0.07, 20, 12);
      const leftEye = new THREE.Mesh(eyeGeometry, new THREE.MeshStandardMaterial({ color: 0x7df9ff, emissive: 0x1abbd1, emissiveIntensity: 0.7 }));
      leftEye.position.set(-0.26, 0.5, 0.78);
      const rightEye = leftEye.clone();
      rightEye.position.x = 0.26;
      group.add(leftEye, rightEye);

      const smile = new THREE.Mesh(
        new THREE.TorusGeometry(0.16, 0.015, 8, 32, Math.PI),
        new THREE.MeshStandardMaterial({ color: 0x7df9ff, emissive: 0x1abbd1, emissiveIntensity: 0.55 }),
      );
      smile.position.set(0, 0.34, 0.78);
      smile.rotation.z = Math.PI;
      group.add(smile);

      const cheekGeometry = new THREE.SphereGeometry(0.08, 20, 12);
      const leftCheek = new THREE.Mesh(cheekGeometry, cheekMaterial);
      leftCheek.scale.set(1, 0.55, 0.28);
      leftCheek.position.set(-0.48, 0.34, 0.75);
      const rightCheek = leftCheek.clone();
      rightCheek.position.x = 0.48;
      group.add(leftCheek, rightCheek);

      const earGeometry = new THREE.SphereGeometry(0.18, 24, 16);
      const leftEar = new THREE.Mesh(earGeometry, shellMaterial);
      leftEar.scale.set(0.75, 1, 0.7);
      leftEar.position.set(-0.82, 0.46, 0.03);
      const rightEar = leftEar.clone();
      rightEar.position.x = 0.82;
      group.add(leftEar, rightEar);

      const antennaStem = new THREE.Mesh(
        new THREE.CylinderGeometry(0.025, 0.025, 0.46, 14),
        glowMaterial,
      );
      antennaStem.position.y = 1.24;
      antennaStem.rotation.z = Math.sin(0.35) * 0.18;
      group.add(antennaStem);

      const antennaDot = new THREE.Mesh(
        new THREE.SphereGeometry(0.1, 24, 16),
        glowMaterial,
      );
      antennaDot.position.y = 1.5;
      group.add(antennaDot);

      const armGeometry = new THREE.CapsuleGeometry(0.08, 0.52, 10, 18);
      const leftArm = new THREE.Mesh(armGeometry, shellMaterial);
      leftArm.position.set(-0.72, -0.35, 0.08);
      leftArm.rotation.z = 0.46;
      const rightArm = leftArm.clone();
      rightArm.position.x = 0.72;
      rightArm.rotation.z = -0.46;
      group.add(leftArm, rightArm);

      const handGeometry = new THREE.SphereGeometry(0.12, 20, 12);
      const leftHand = new THREE.Mesh(handGeometry, shellMaterial);
      leftHand.position.set(-0.9, -0.7, 0.12);
      const rightHand = leftHand.clone();
      rightHand.position.x = 0.9;
      group.add(leftHand, rightHand);

      const footGeometry = new THREE.SphereGeometry(0.18, 24, 12);
      const leftFoot = new THREE.Mesh(footGeometry, darkMaterial);
      leftFoot.scale.set(1.3, 0.42, 0.7);
      leftFoot.position.set(-0.33, -1.18, 0.1);
      const rightFoot = leftFoot.clone();
      rightFoot.position.x = 0.33;
      group.add(leftFoot, rightFoot);

      const halo = new THREE.Mesh(
        new THREE.TorusGeometry(1.03, 0.025, 12, 96),
        glowMaterial,
      );
      halo.position.y = 0.08;
      halo.rotation.x = Math.PI / 2;
      group.add(halo);

      const shadow = new THREE.Mesh(
        new THREE.CircleGeometry(1.0, 64),
        new THREE.MeshBasicMaterial({ color: 0x8fb5ff, transparent: true, opacity: 0.28 }),
      );
      shadow.position.y = -1.35;
      shadow.rotation.x = -Math.PI / 2;
      group.add(shadow);

      let frame = 0;
      let animationId = 0;
      const animate = () => {
        frame += 0.018;
        group.rotation.y = Math.sin(frame) * 0.18;
        group.position.y = Math.sin(frame * 1.7) * 0.04;
        head.rotation.z = Math.sin(frame * 1.3) * 0.035;
        leftArm.rotation.z = 0.46 + Math.sin(frame * 2.1) * 0.08;
        rightArm.rotation.z = -0.46 + Math.cos(frame * 2.1) * 0.08;
        antennaDot.scale.setScalar(1 + Math.sin(frame * 4.4) * (thinking ? 0.18 : 0.08));
        halo.rotation.z += thinking ? 0.035 : 0.012;
        const pulse = active || thinking ? 0.08 : 0.03;
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
