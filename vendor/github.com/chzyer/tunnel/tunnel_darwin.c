#include <sys/sys_domain.h>
#include <sys/kern_control.h>
#include <net/if_utun.h>
#include <string.h>
#include <sys/socket.h>
#include <sys/ioctl.h>
#include <stdio.h>
#include <unistd.h>
#include <errno.h>

static int utun_open(int idx) {
	struct ctl_info ctlInfo;
	strlcpy(ctlInfo.ctl_name, UTUN_CONTROL_NAME, sizeof(ctlInfo.ctl_name));
		int fd;
	fd = socket(PF_SYSTEM, SOCK_DGRAM, SYSPROTO_CONTROL);
	if (fd < 0) {
		return -errno;
	}
	
	struct sockaddr_ctl sc;

	if (ioctl(fd, CTLIOCGINFO, &ctlInfo) == -1) {
		close(fd);
		return -errno;
	}

	sc.sc_id = ctlInfo.ctl_id;
	sc.sc_len = sizeof(sc);
	sc.sc_family = AF_SYSTEM;
	sc.ss_sysaddr = AF_SYS_CONTROL;
	sc.sc_unit = idx+1;

	if (connect(fd, (struct sockaddr *)&sc, sizeof(sc)) < 0) {
		close(fd);
		return -errno;
    }

	return fd;
}
