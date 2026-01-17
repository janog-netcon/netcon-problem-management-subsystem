package controllers

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	util "github.com/janog-netcon/netcon-problem-management-subsystem/pkg/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var _ = Describe("ProblemEnvironment controller", func() {
	ctx := context.Background()

	var stopFunc func()

	BeforeEach(func() {
		err := k8sClient.DeleteAllOf(ctx, &netconv1alpha1.Worker{})
		Expect(err).ToNot(HaveOccurred())
		err = k8sClient.DeleteAllOf(ctx, &netconv1alpha1.Problem{}, client.InNamespace("default"))
		Expect(err).ToNot(HaveOccurred())
		err = k8sClient.DeleteAllOf(ctx, &netconv1alpha1.ProblemEnvironment{}, client.InNamespace("default"))
		Expect(err).ToNot(HaveOccurred())
		time.Sleep(100 * time.Millisecond)

		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme: scheme.Scheme,
			Metrics: server.Options{
				BindAddress: "0",
			},
		})
		Expect(err).ToNot(HaveOccurred())

		err = (&ProblemEnvironmentReconciler{
			Client:   k8sClient,
			Scheme:   scheme.Scheme,
			Recorder: mgr.GetEventRecorderFor("problemenvironment-controller"),
		}).SetupWithManager(mgr)
		Expect(err).NotTo(HaveOccurred())

		ctx, cancel := context.WithCancel(ctx)
		stopFunc = cancel
		go func() {
			err := mgr.Start(ctx)
			if err != nil {
				panic(err)
			}
		}()
		time.Sleep(100 * time.Millisecond)
	})

	AfterEach(func() {
		stopFunc()
		time.Sleep(100 * time.Millisecond)
	})

	It("should confirm worker if valid workerName is specified", func() {
		worker001 := netconv1alpha1.Worker{}
		worker001.Name = "worker-001"

		problemEnvironment := netconv1alpha1.ProblemEnvironment{}
		err := loadManifest(
			filepath.Join("tests", "problemenvironments", "problemenvironment-tst-001.yaml"),
			&problemEnvironment,
		)
		Expect(err).NotTo(HaveOccurred())

		namespace := problemEnvironment.Namespace
		name := problemEnvironment.Name

		err = k8sClient.Create(ctx, &worker001)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(100 * time.Millisecond)

		err = k8sClient.Create(ctx, &problemEnvironment)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(100 * time.Millisecond)

		Eventually(func() error {
			problemEnvironment := netconv1alpha1.ProblemEnvironment{}
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Namespace: namespace,
				Name:      name,
			}, &problemEnvironment); err != nil {
				return err
			}

			if util.GetProblemEnvironmentCondition(
				&problemEnvironment,
				netconv1alpha1.ProblemEnvironmentConditionScheduled,
			) != metav1.ConditionTrue {
				return fmt.Errorf("failed to confirm schedule")
			}

			return nil
		}).ShouldNot(HaveOccurred())
	})

	It("should not confirm worker if invalid workerName is specified", func() {
		problemEnvironment := netconv1alpha1.ProblemEnvironment{}
		err := loadManifest(
			filepath.Join("tests", "problemenvironments", "problemenvironment-tst-001.yaml"),
			&problemEnvironment,
		)
		Expect(err).NotTo(HaveOccurred())

		namespace := problemEnvironment.Namespace
		name := problemEnvironment.Name

		err = k8sClient.Create(ctx, &problemEnvironment)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(100 * time.Millisecond)

		Eventually(func() error {
			problemEnvironment := netconv1alpha1.ProblemEnvironment{}
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Namespace: namespace,
				Name:      name,
			}, &problemEnvironment); err != nil {
				return err
			}

			if util.GetProblemEnvironmentCondition(
				&problemEnvironment,
				netconv1alpha1.ProblemEnvironmentConditionScheduled,
			) != metav1.ConditionFalse {
				return fmt.Errorf("failed to confirm schedule")
			}

			return nil
		}).ShouldNot(HaveOccurred())
	})

	It("should not schedule worker if there is no schedulable workers", func() {
		worker001 := netconv1alpha1.Worker{}
		worker001.Name = "worker-001"
		worker001.Spec.DisableSchedule = true

		problemEnvironment := netconv1alpha1.ProblemEnvironment{}
		err := loadManifest(
			filepath.Join("tests", "problemenvironments", "problemenvironment-tst-002.yaml"),
			&problemEnvironment,
		)
		Expect(err).NotTo(HaveOccurred())

		namespace := problemEnvironment.Namespace
		name := problemEnvironment.Name

		err = k8sClient.Create(ctx, &worker001)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(100 * time.Millisecond)

		err = k8sClient.Get(ctx, types.NamespacedName{Name: "worker-001"}, &worker001)
		Expect(err).NotTo(HaveOccurred())
		util.SetWorkerCondition(
			&worker001,
			netconv1alpha1.WorkerConditionReady,
			metav1.ConditionTrue,
			"Test", "test",
		)
		worker001.Status.WorkerInfo.CPUUsedPercent = "50.0"
		worker001.Status.WorkerInfo.MemoryUsedPercent = "90.0"
		err = k8sClient.Status().Update(ctx, &worker001)
		Expect(err).NotTo(HaveOccurred())

		err = k8sClient.Create(ctx, &problemEnvironment)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(100 * time.Millisecond)

		Eventually(func() error {
			problemEnvironment := netconv1alpha1.ProblemEnvironment{}
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Namespace: namespace,
				Name:      name,
			}, &problemEnvironment); err != nil {
				return err
			}

			if util.GetProblemEnvironmentCondition(
				&problemEnvironment,
				netconv1alpha1.ProblemEnvironmentConditionScheduled,
			) != metav1.ConditionFalse {
				return fmt.Errorf("failed to confirm schedule")
			}

			return nil
		}).ShouldNot(HaveOccurred())
	})

	It("should schedule ProblemEnvironment to the worker which uses less resources", func() {
		worker001 := netconv1alpha1.Worker{}
		worker001.Name = "worker-001"

		worker002 := netconv1alpha1.Worker{}
		worker002.Name = "worker-002"

		problemEnvironment := netconv1alpha1.ProblemEnvironment{}
		err := loadManifest(
			filepath.Join("tests", "problemenvironments", "problemenvironment-tst-002.yaml"),
			&problemEnvironment,
		)
		Expect(err).NotTo(HaveOccurred())

		namespace := problemEnvironment.Namespace
		name := problemEnvironment.Name

		err = k8sClient.Create(ctx, &worker001)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(100 * time.Millisecond)

		err = k8sClient.Create(ctx, &worker002)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(100 * time.Millisecond)

		err = k8sClient.Get(ctx, types.NamespacedName{Name: "worker-001"}, &worker001)
		Expect(err).NotTo(HaveOccurred())
		util.SetWorkerCondition(
			&worker001,
			netconv1alpha1.WorkerConditionReady,
			metav1.ConditionTrue,
			"Test", "test",
		)
		worker001.Status.WorkerInfo.CPUUsedPercent = "50.0"
		worker001.Status.WorkerInfo.MemoryUsedPercent = "90.0"
		err = k8sClient.Status().Update(ctx, &worker001)
		Expect(err).NotTo(HaveOccurred())

		err = k8sClient.Get(ctx, types.NamespacedName{Name: "worker-002"}, &worker002)
		Expect(err).NotTo(HaveOccurred())
		util.SetWorkerCondition(
			&worker002,
			netconv1alpha1.WorkerConditionReady,
			metav1.ConditionTrue,
			"Test", "test",
		)
		worker002.Status.WorkerInfo.CPUUsedPercent = "10.0"
		worker002.Status.WorkerInfo.MemoryUsedPercent = "30.0"
		err = k8sClient.Status().Update(ctx, &worker002)
		Expect(err).NotTo(HaveOccurred())

		err = k8sClient.Create(ctx, &problemEnvironment)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(100 * time.Millisecond)

		Eventually(func() error {
			problemEnvironment := netconv1alpha1.ProblemEnvironment{}
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Namespace: namespace,
				Name:      name,
			}, &problemEnvironment); err != nil {
				return err
			}

			if problemEnvironment.Spec.WorkerName != "worker-002" {
				return fmt.Errorf("invalid scheduling")
			}

			if util.GetProblemEnvironmentCondition(
				&problemEnvironment,
				netconv1alpha1.ProblemEnvironmentConditionScheduled,
			) != metav1.ConditionTrue {
				return fmt.Errorf("failed to confirm schedule")
			}

			return nil
		}).ShouldNot(HaveOccurred())
	})

	It("should not schedule ProblemEnvironment to the worker which is disabled", func() {
		worker002 := netconv1alpha1.Worker{}
		worker002.Name = "worker-002"

		worker003 := netconv1alpha1.Worker{}
		worker003.Name = "worker-003"
		worker003.Spec.DisableSchedule = true

		problemEnvironment := netconv1alpha1.ProblemEnvironment{}
		err := loadManifest(
			filepath.Join("tests", "problemenvironments", "problemenvironment-tst-002.yaml"),
			&problemEnvironment,
		)
		Expect(err).NotTo(HaveOccurred())

		namespace := problemEnvironment.Namespace
		name := problemEnvironment.Name

		err = k8sClient.Create(ctx, &worker002)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(100 * time.Millisecond)

		err = k8sClient.Create(ctx, &worker003)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(100 * time.Millisecond)

		err = k8sClient.Get(ctx, types.NamespacedName{Name: "worker-002"}, &worker002)
		Expect(err).NotTo(HaveOccurred())
		util.SetWorkerCondition(
			&worker002,
			netconv1alpha1.WorkerConditionReady,
			metav1.ConditionTrue,
			"Test", "test",
		)
		worker002.Status.WorkerInfo.CPUUsedPercent = "50.0"
		worker002.Status.WorkerInfo.MemoryUsedPercent = "90.0"
		err = k8sClient.Status().Update(ctx, &worker002)
		Expect(err).NotTo(HaveOccurred())

		err = k8sClient.Get(ctx, types.NamespacedName{Name: "worker-003"}, &worker003)
		Expect(err).NotTo(HaveOccurred())
		util.SetWorkerCondition(
			&worker003,
			netconv1alpha1.WorkerConditionReady,
			metav1.ConditionTrue,
			"Test", "test",
		)
		worker003.Status.WorkerInfo.CPUUsedPercent = "10.0"
		worker003.Status.WorkerInfo.MemoryUsedPercent = "30.0"
		err = k8sClient.Status().Update(ctx, &worker003)
		Expect(err).NotTo(HaveOccurred())

		err = k8sClient.Create(ctx, &problemEnvironment)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(100 * time.Millisecond)

		Eventually(func() error {
			problemEnvironment := netconv1alpha1.ProblemEnvironment{}
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Namespace: namespace,
				Name:      name,
			}, &problemEnvironment); err != nil {
				return err
			}

			if problemEnvironment.Spec.WorkerName != "worker-002" {
				return fmt.Errorf("invalid scheduling")
			}

			if util.GetProblemEnvironmentCondition(
				&problemEnvironment,
				netconv1alpha1.ProblemEnvironmentConditionScheduled,
			) != metav1.ConditionTrue {
				return fmt.Errorf("failed to confirm schedule")
			}

			return nil
		}).ShouldNot(HaveOccurred())
	})

	It("should reflect container status to condition Ready ", func() {
		worker001 := netconv1alpha1.Worker{}
		worker001.Name = "worker-001"

		problemEnvironment := netconv1alpha1.ProblemEnvironment{}
		err := loadManifest(
			filepath.Join("tests", "problemenvironments", "problemenvironment-tst-002.yaml"),
			&problemEnvironment,
		)
		Expect(err).NotTo(HaveOccurred())

		namespace := problemEnvironment.Namespace
		name := problemEnvironment.Name

		err = k8sClient.Create(ctx, &worker001)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(100 * time.Millisecond)

		err = k8sClient.Get(ctx, types.NamespacedName{Name: "worker-001"}, &worker001)
		Expect(err).NotTo(HaveOccurred())
		util.SetWorkerCondition(
			&worker001,
			netconv1alpha1.WorkerConditionReady,
			metav1.ConditionTrue,
			"Test", "test",
		)
		worker001.Status.WorkerInfo.CPUUsedPercent = "10.0"
		worker001.Status.WorkerInfo.MemoryUsedPercent = "30.0"
		err = k8sClient.Status().Update(ctx, &worker001)
		Expect(err).NotTo(HaveOccurred())

		err = k8sClient.Create(ctx, &problemEnvironment)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(100 * time.Millisecond)

		Eventually(func() error {
			problemEnvironment := netconv1alpha1.ProblemEnvironment{}
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Namespace: namespace,
				Name:      name,
			}, &problemEnvironment); err != nil {
				return err
			}

			if problemEnvironment.Spec.WorkerName != "worker-001" {
				return fmt.Errorf("invalid scheduling")
			}

			if util.GetProblemEnvironmentCondition(
				&problemEnvironment,
				netconv1alpha1.ProblemEnvironmentConditionScheduled,
			) != metav1.ConditionTrue {
				return fmt.Errorf("failed to confirm schedule")
			}

			return nil
		}).ShouldNot(HaveOccurred())

		err = k8sClient.Get(ctx, types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}, &problemEnvironment)
		Expect(err).NotTo(HaveOccurred())

		problemEnvironment.Status.Containers = []netconv1alpha1.ContainerStatus{
			{
				Ready: false,
			},
			{
				Ready: true,
			},
		}
		err = k8sClient.Status().Update(ctx, &problemEnvironment)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(100 * time.Millisecond)

		Eventually(func() error {
			problemEnvironment := netconv1alpha1.ProblemEnvironment{}
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Namespace: namespace,
				Name:      name,
			}, &problemEnvironment); err != nil {
				return err
			}

			if util.GetProblemEnvironmentCondition(
				&problemEnvironment,
				netconv1alpha1.ProblemEnvironmentConditionReady,
			) != metav1.ConditionFalse {
				return fmt.Errorf("failed to confirm schedule")
			}

			return nil
		}).ShouldNot(HaveOccurred())

		err = k8sClient.Get(ctx, types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}, &problemEnvironment)
		Expect(err).NotTo(HaveOccurred())

		problemEnvironment.Status.Containers = []netconv1alpha1.ContainerStatus{
			{
				Ready: true,
			},
			{
				Ready: true,
			},
		}
		err = k8sClient.Status().Update(ctx, &problemEnvironment)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(100 * time.Millisecond)

		Eventually(func() error {
			problemEnvironment := netconv1alpha1.ProblemEnvironment{}
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Namespace: namespace,
				Name:      name,
			}, &problemEnvironment); err != nil {
				return err
			}

			if util.GetProblemEnvironmentCondition(
				&problemEnvironment,
				netconv1alpha1.ProblemEnvironmentConditionReady,
			) != metav1.ConditionTrue {
				return fmt.Errorf("failed to confirm schedule")
			}

			return nil
		}).ShouldNot(HaveOccurred())
	})

	Describe("with WorkerSelectors", func() {
		It("should schedule ProblemEnvironment to the worker which matches workerSelectors", func() {
			worker001 := netconv1alpha1.Worker{}
			worker001.Name = "worker-001"
			worker001.Labels = map[string]string{"class": "foo"}

			worker002 := netconv1alpha1.Worker{}
			worker002.Name = "worker-002"
			worker002.Labels = map[string]string{"class": "bar"}

			problemEnvironment := netconv1alpha1.ProblemEnvironment{}
			err := loadManifest(
				filepath.Join("tests", "problemenvironments", "problemenvironment-tst-002.yaml"),
				&problemEnvironment,
			)
			Expect(err).NotTo(HaveOccurred())

			// Add WorkerSelectors
			problemEnvironment.Spec.WorkerSelectors = []metav1.LabelSelector{
				{
					MatchLabels: map[string]string{"class": "bar"},
				},
			}

			namespace := problemEnvironment.Namespace
			name := problemEnvironment.Name

			err = k8sClient.Create(ctx, &worker001)
			Expect(err).NotTo(HaveOccurred())
			time.Sleep(100 * time.Millisecond)

			err = k8sClient.Create(ctx, &worker002)
			Expect(err).NotTo(HaveOccurred())
			time.Sleep(100 * time.Millisecond)

			err = k8sClient.Get(ctx, types.NamespacedName{Name: "worker-001"}, &worker001)
			Expect(err).NotTo(HaveOccurred())
			util.SetWorkerCondition(
				&worker001,
				netconv1alpha1.WorkerConditionReady,
				metav1.ConditionTrue,
				"Test", "test",
			)
			worker001.Status.WorkerInfo.CPUUsedPercent = "10.0"
			worker001.Status.WorkerInfo.MemoryUsedPercent = "30.0"
			err = k8sClient.Status().Update(ctx, &worker001)
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Get(ctx, types.NamespacedName{Name: "worker-002"}, &worker002)
			Expect(err).NotTo(HaveOccurred())
			util.SetWorkerCondition(
				&worker002,
				netconv1alpha1.WorkerConditionReady,
				metav1.ConditionTrue,
				"Test", "test",
			)
			worker002.Status.WorkerInfo.CPUUsedPercent = "5.0"
			worker002.Status.WorkerInfo.MemoryUsedPercent = "10.0"
			err = k8sClient.Status().Update(ctx, &worker002)
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Create(ctx, &problemEnvironment)
			Expect(err).NotTo(HaveOccurred())
			time.Sleep(100 * time.Millisecond)

			Eventually(func() error {
				problemEnvironment := netconv1alpha1.ProblemEnvironment{}
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Namespace: namespace,
					Name:      name,
				}, &problemEnvironment); err != nil {
					return err
				}

				if problemEnvironment.Spec.WorkerName != "worker-002" {
					return fmt.Errorf("invalid scheduling: expected worker-002, got %s", problemEnvironment.Spec.WorkerName)
				}

				if util.GetProblemEnvironmentCondition(
					&problemEnvironment,
					netconv1alpha1.ProblemEnvironmentConditionScheduled,
				) != metav1.ConditionTrue {
					return fmt.Errorf("failed to confirm schedule")
				}

				return nil
			}).ShouldNot(HaveOccurred())
		})

		It("should not schedule ProblemEnvironment if no worker matches workerSelectors", func() {
			worker001 := netconv1alpha1.Worker{}
			worker001.Name = "worker-001"
			worker001.Labels = map[string]string{"class": "foo"}

			problemEnvironment := netconv1alpha1.ProblemEnvironment{}
			err := loadManifest(
				filepath.Join("tests", "problemenvironments", "problemenvironment-tst-002.yaml"),
				&problemEnvironment,
			)
			Expect(err).NotTo(HaveOccurred())

			// Add WorkerSelectors
			problemEnvironment.Spec.WorkerSelectors = []metav1.LabelSelector{
				{
					MatchLabels: map[string]string{"class": "bar"},
				},
			}

			namespace := problemEnvironment.Namespace
			name := problemEnvironment.Name

			err = k8sClient.Create(ctx, &worker001)
			Expect(err).NotTo(HaveOccurred())
			time.Sleep(100 * time.Millisecond)

			err = k8sClient.Get(ctx, types.NamespacedName{Name: "worker-001"}, &worker001)
			Expect(err).NotTo(HaveOccurred())
			util.SetWorkerCondition(
				&worker001,
				netconv1alpha1.WorkerConditionReady,
				metav1.ConditionTrue,
				"Test", "test",
			)
			worker001.Status.WorkerInfo.CPUUsedPercent = "10.0"
			worker001.Status.WorkerInfo.MemoryUsedPercent = "30.0"
			err = k8sClient.Status().Update(ctx, &worker001)
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Create(ctx, &problemEnvironment)
			Expect(err).NotTo(HaveOccurred())
			time.Sleep(100 * time.Millisecond)

			Consistently(func() error {
				problemEnvironment := netconv1alpha1.ProblemEnvironment{}
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Namespace: namespace,
					Name:      name,
				}, &problemEnvironment); err != nil {
					return err
				}

				if problemEnvironment.Spec.WorkerName != "" {
					return fmt.Errorf("should not be scheduled, but got %s", problemEnvironment.Spec.WorkerName)
				}

				if util.GetProblemEnvironmentCondition(
					&problemEnvironment,
					netconv1alpha1.ProblemEnvironmentConditionScheduled,
				) != metav1.ConditionFalse {
					return fmt.Errorf("failed to confirm schedule condition")
				}

				return nil
			}, 3*time.Second, 100*time.Millisecond).Should(Succeed())
		})
	})
})
